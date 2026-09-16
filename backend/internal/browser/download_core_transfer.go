package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const coreDownloadRetries = 6

func prepareCoreRequest(ctx context.Context, targetURL string, start, end int64) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OneBrowser-CoreDownloader/1.8.10")
	req.Header.Set("Accept", "application/octet-stream")
	if start >= 0 {
		value := fmt.Sprintf("bytes=%d-", start)
		if end >= start {
			value = fmt.Sprintf("bytes=%d-%d", start, end)
		}
		req.Header.Set("Range", value)
	}
	return req, nil
}

func retryCoreDelay(ctx context.Context, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(attempt+1) * 650 * time.Millisecond):
		return nil
	}
}

func probeCoreDownload(ctx context.Context, client *http.Client, targetURL string) (int64, bool, error) {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		req, err := prepareCoreRequest(ctx, targetURL, 0, 0)
		if err != nil {
			return 0, false, err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if err = retryCoreDelay(ctx, attempt); err != nil {
				return 0, false, err
			}
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			return 0, false, fmt.Errorf("HTTP状态码异常: %d", resp.StatusCode)
		}
		totalSize, supportRange := resp.ContentLength, resp.StatusCode == http.StatusPartialContent
		if supportRange {
			parts := strings.Split(resp.Header.Get("Content-Range"), "/")
			if len(parts) == 2 {
				if parsed, parseErr := strconv.ParseInt(parts[1], 10, 64); parseErr == nil {
					totalSize = parsed
				}
			}
		}
		return totalSize, supportRange, nil
	}
	return 0, false, fmt.Errorf("连接内核发布源失败，已重试: %w", lastErr)
}

func doConcurrentDownload(ctx context.Context, client *http.Client, targetURL string, tempFile *os.File, sendEvent func(string, int, string)) error {
	totalSize, supportRange, err := probeCoreDownload(ctx, client, targetURL)
	if err != nil {
		return err
	}
	if totalSize <= 0 || !supportRange {
		sendEvent("downloading", 0, "服务器不支持分段下载，使用断点续传模式...")
		return doSingleThreadDownload(ctx, client, targetURL, tempFile, totalSize, sendEvent)
	}
	sendEvent("downloading", 0, fmt.Sprintf("正在并行下载，总大小 %.2f MB", float64(totalSize)/1024/1024))
	if err := tempFile.Truncate(totalSize); err != nil {
		return err
	}
	numWorkers := 8
	if totalSize < 8*1024*1024 {
		numWorkers = 4
	}
	chunkSize := totalSize / int64(numWorkers)
	var wg sync.WaitGroup
	var downloaded int64
	var mu sync.Mutex
	var lastTick time.Time
	var downloadErr error
	report := func(n int64) {
		mu.Lock()
		downloaded += n
		if time.Since(lastTick) > time.Second {
			percent := int(float64(downloaded) / float64(totalSize) * 100)
			if percent > 99 {
				percent = 99
			}
			sendEvent("downloading", percent, fmt.Sprintf("并行下载中... %.2f MB / %.2f MB", float64(downloaded)/1024/1024, float64(totalSize)/1024/1024))
			lastTick = time.Now()
		}
		mu.Unlock()
	}
	for i := 0; i < numWorkers; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == numWorkers-1 {
			end = totalSize - 1
		}
		wg.Add(1)
		go func(start, end int64) {
			defer wg.Done()
			current := start
			var lastErr error
			for attempt := 0; attempt < coreDownloadRetries && current <= end; attempt++ {
				if ctx.Err() != nil {
					return
				}
				req, err := prepareCoreRequest(ctx, targetURL, current, end)
				if err != nil {
					lastErr = err
					break
				}
				resp, err := client.Do(req)
				if err != nil {
					lastErr = err
					if retryCoreDelay(ctx, attempt) != nil {
						return
					}
					continue
				}
				if resp.StatusCode != http.StatusPartialContent {
					resp.Body.Close()
					lastErr = fmt.Errorf("分段请求返回 HTTP %d", resp.StatusCode)
					break
				}
				buf := make([]byte, 256*1024)
				for current <= end {
					n, readErr := resp.Body.Read(buf)
					if n > 0 {
						remaining := end - current + 1
						if int64(n) > remaining {
							n = int(remaining)
						}
						written, writeErr := tempFile.WriteAt(buf[:n], current)
						if writeErr != nil || written != n {
							if writeErr == nil {
								writeErr = io.ErrShortWrite
							}
							lastErr = writeErr
							break
						}
						current += int64(written)
						report(int64(written))
					}
					if readErr != nil {
						if readErr == io.EOF && current > end {
							lastErr = nil
						} else if readErr == io.EOF {
							lastErr = io.ErrUnexpectedEOF
						} else {
							lastErr = readErr
						}
						break
					}
				}
				resp.Body.Close()
				if current > end {
					return
				}
				if retryCoreDelay(ctx, attempt) != nil {
					return
				}
			}
			mu.Lock()
			if downloadErr == nil {
				downloadErr = fmt.Errorf("分段 %d-%d 下载中断，已自动重试: %w", start, end, lastErr)
			}
			mu.Unlock()
		}(start, end)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	mu.Lock()
	err = downloadErr
	complete := downloaded == totalSize
	mu.Unlock()
	if err == nil && complete {
		sendEvent("downloading", 100, "下载完成，正在校验...")
		return nil
	}
	sendEvent("downloading", 0, "分段连接不稳定，已切换断点续传模式...")
	return doSingleThreadDownload(ctx, client, targetURL, tempFile, totalSize, sendEvent)
}

func doSingleThreadDownload(ctx context.Context, client *http.Client, targetURL string, tempFile *os.File, totalSize int64, sendEvent func(string, int, string)) error {
	if err := tempFile.Truncate(0); err != nil {
		return err
	}
	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var downloaded int64
	var lastTick time.Time
	var lastErr error
	for attempt := 0; attempt < coreDownloadRetries; attempt++ {
		start := int64(-1)
		if downloaded > 0 {
			start = downloaded
		}
		req, err := prepareCoreRequest(ctx, targetURL, start, -1)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if err = retryCoreDelay(ctx, attempt); err != nil {
				return err
			}
			continue
		}
		if downloaded > 0 && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			downloaded = 0
			if err := tempFile.Truncate(0); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			return fmt.Errorf("HTTP状态码异常: %d", resp.StatusCode)
		}
		if totalSize <= 0 {
			if resp.StatusCode == http.StatusOK {
				totalSize = resp.ContentLength
			} else if parts := strings.Split(resp.Header.Get("Content-Range"), "/"); len(parts) == 2 {
				totalSize, _ = strconv.ParseInt(parts[1], 10, 64)
			}
		}
		buf := make([]byte, 1024*1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				written, writeErr := tempFile.WriteAt(buf[:n], downloaded)
				if writeErr != nil || written != n {
					resp.Body.Close()
					if writeErr == nil {
						writeErr = io.ErrShortWrite
					}
					return writeErr
				}
				downloaded += int64(written)
				if totalSize > 0 && time.Since(lastTick) > time.Second {
					percent := int(float64(downloaded) / float64(totalSize) * 100)
					if percent > 99 {
						percent = 99
					}
					sendEvent("downloading", percent, fmt.Sprintf("断点续传中... %.2f MB / %.2f MB", float64(downloaded)/1024/1024, float64(totalSize)/1024/1024))
					lastTick = time.Now()
				}
			}
			if readErr != nil {
				resp.Body.Close()
				if readErr == io.EOF && (totalSize <= 0 || downloaded == totalSize) {
					sendEvent("downloading", 100, "下载完成，正在校验...")
					return nil
				}
				if readErr == io.EOF {
					lastErr = io.ErrUnexpectedEOF
				} else {
					lastErr = readErr
				}
				break
			}
		}
		if err := retryCoreDelay(ctx, attempt); err != nil {
			return err
		}
	}
	return fmt.Errorf("下载连接反复中断，已自动重试并保留续传进度: %w", lastErr)
}
