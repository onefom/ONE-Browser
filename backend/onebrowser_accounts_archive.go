package backend

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const accountArchiveLimit = 16 << 20

var accountRARSignature = []byte{'R', 'a', 'r', '!', 0x1a, 7, 1, 0}

type OneBrowserAccountRecord struct {
	AccountID string `json:"accountId"`
	Service   string `json:"service"`
	Identity  string `json:"identity"`
	Owner     string `json:"owner"`
}
type oneBrowserAccountPackage struct {
	Format   string                    `json:"format"`
	Version  int                       `json:"version"`
	Accounts []OneBrowserAccountRecord `json:"accounts"`
}

func validateAccountRecords(items []OneBrowserAccountRecord) error {
	if len(items) > 5000 {
		return fmt.Errorf("账号数量不能超过 5000")
	}
	for _, a := range items {
		if strings.TrimSpace(a.Service) == "" || utf8.RuneCountInString(a.Service) > 120 || len(a.Identity) > 4096 || utf8.RuneCountInString(a.Owner) > 500 || len(a.AccountID) > 128 {
			return fmt.Errorf("账号资料格式无效，请检查名称、密码或备注长度")
		}
	}
	return nil
}
func accountRARVint(n uint64) []byte {
	var b [10]byte
	size := binary.PutUvarint(b[:], n)
	return append([]byte(nil), b[:size]...)
}
func accountRARHeader(body []byte) []byte {
	data := append(accountRARVint(uint64(len(body))), body...)
	var c [4]byte
	binary.LittleEndian.PutUint32(c[:], crc32.ChecksumIEEE(data))
	return append(c[:], data...)
}

// RAR5 storage method 0: a genuine RAR archive, without external software.
// Only the fixed accounts.json entry is written; no filesystem paths are imported.
func encodeAccountRAR(payload []byte) []byte {
	out := append([]byte(nil), accountRARSignature...)
	out = append(out, accountRARHeader([]byte{1, 0, 0})...)
	body := []byte{2, 2}
	body = append(body, accountRARVint(uint64(len(payload)))...)
	body = append(body, 4)
	body = append(body, accountRARVint(uint64(len(payload)))...)
	body = append(body, 0x20)
	var c [4]byte
	binary.LittleEndian.PutUint32(c[:], crc32.ChecksumIEEE(payload))
	body = append(body, c[:]...)
	body = append(body, 0, 0, 13)
	body = append(body, []byte("accounts.json")...)
	out = append(out, accountRARHeader(body)...)
	out = append(out, payload...)
	return append(out, accountRARHeader([]byte{5, 0, 0})...)
}
func decodeAccountRAR(data []byte) ([]byte, error) {
	fail := func() ([]byte, error) {
		return nil, fmt.Errorf("账号 RAR 文件损坏或不是本程序导出的格式")
	}
	if len(data) > accountArchiveLimit || !bytes.HasPrefix(data, accountRARSignature) {
		return fail()
	}
	pos := len(accountRARSignature)
	var payload []byte
	mainSeen := false
	for pos < len(data) {
		if len(data)-pos < 5 {
			return fail()
		}
		want := binary.LittleEndian.Uint32(data[pos:])
		start := pos + 4
		n, k := binary.Uvarint(data[start:])
		if k <= 0 || k > 3 || n > 2<<20 || n > uint64(len(data)-start-k) {
			return fail()
		}
		bodyStart := start + k
		end := bodyStart + int(n)
		if crc32.ChecksumIEEE(data[start:end]) != want {
			return fail()
		}
		b := bytes.NewReader(data[bodyStart:end])
		typ, e := binary.ReadUvarint(b)
		if e != nil {
			return fail()
		}
		flags, e := binary.ReadUvarint(b)
		if e != nil || flags&^uint64(2) != 0 {
			return fail()
		}
		size := uint64(0)
		if flags&2 != 0 {
			size, e = binary.ReadUvarint(b)
			if e != nil || size > uint64(len(data)-end) {
				return fail()
			}
		}
		switch typ {
		case 1:
			if mainSeen || payload != nil {
				return fail()
			}
			v, e := binary.ReadUvarint(b)
			if e != nil || v != 0 || b.Len() != 0 {
				return fail()
			}
			mainSeen = true
		case 2:
			if !mainSeen || payload != nil {
				return fail()
			}
			ff, e := binary.ReadUvarint(b)
			if e != nil || ff != 4 {
				return fail()
			}
			unpacked, e := binary.ReadUvarint(b)
			if e != nil || unpacked != size {
				return fail()
			}
			if _, e = binary.ReadUvarint(b); e != nil {
				return fail()
			}
			var c [4]byte
			if _, e = io.ReadFull(b, c[:]); e != nil {
				return fail()
			}
			compression, e := binary.ReadUvarint(b)
			if e != nil || compression != 0 {
				return fail()
			}
			host, e := binary.ReadUvarint(b)
			if e != nil || host != 0 {
				return fail()
			}
			length, e := binary.ReadUvarint(b)
			if e != nil || length != 13 || b.Len() != 13 {
				return fail()
			}
			name := make([]byte, 13)
			io.ReadFull(b, name)
			if string(name) != "accounts.json" {
				return fail()
			}
			payload = append([]byte{}, data[end:end+int(size)]...)
			if crc32.ChecksumIEEE(payload) != binary.LittleEndian.Uint32(c[:]) {
				return fail()
			}
		case 5:
			v, e := binary.ReadUvarint(b)
			if e != nil || v != 0 || b.Len() != 0 || payload == nil || size != 0 || end != len(data) {
				return fail()
			}
			return payload, nil
		default:
			return fail()
		}
		pos = end + int(size)
	}
	return fail()
}
func decodeAccountPackage(data []byte) ([]OneBrowserAccountRecord, error) {
	if len(data) > accountArchiveLimit {
		return nil, fmt.Errorf("账号文件不能超过 16MB")
	}
	if bytes.HasPrefix(data, accountRARSignature) {
		var err error
		data, err = decodeAccountRAR(data)
		if err != nil {
			return nil, err
		}
	}
	var p oneBrowserAccountPackage
	if err := json.Unmarshal(data, &p); err != nil || p.Format != "OneBrowserAccounts" || p.Version != 1 || p.Accounts == nil {
		return nil, fmt.Errorf("不是有效的 One Browser 账号文件")
	}
	if err := validateAccountRecords(p.Accounts); err != nil {
		return nil, err
	}
	return p.Accounts, nil
}
func (a *App) OneBrowserExportAccounts(accounts []OneBrowserAccountRecord) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("账号导出服务未就绪")
	}
	if err := validateAccountRecords(accounts); err != nil {
		return "", err
	}
	payload, err := json.MarshalIndent(oneBrowserAccountPackage{"OneBrowserAccounts", 1, accounts}, "", "  ")
	if err != nil {
		return "", err
	}
	if len(payload) > accountArchiveLimit-1024 {
		return "", fmt.Errorf("账号资料不能超过 16MB")
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{Title: "导出账号", DefaultFilename: "OneBrowser-accounts-" + time.Now().Format("20060102-150405") + ".rar", Filters: []wailsruntime.FileFilter{{DisplayName: "RAR 账号归档 (*.rar)", Pattern: "*.rar"}}})
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".rar") {
		path += ".rar"
	}
	return path, os.WriteFile(path, encodeAccountRAR(payload), 0600)
}
func (a *App) OneBrowserImportAccounts() ([]OneBrowserAccountRecord, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("账号导入服务未就绪")
	}
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: "导入账号", Filters: []wailsruntime.FileFilter{{DisplayName: "One Browser 账号 (*.rar;*.json)", Pattern: "*.rar;*.json"}}})
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, accountArchiveLimit+1))
	if err != nil {
		return nil, err
	}
	return decodeAccountPackage(data)
}
