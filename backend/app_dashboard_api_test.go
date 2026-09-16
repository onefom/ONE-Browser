package backend

import (
	"ant-chrome/backend/internal/logger"
	"testing"
)

func TestGetAppLogsReturnsCompleteStreamForLevelFilters(t *testing.T) {
	writer := logger.GetMemoryWriter()
	writer.Clear()
	t.Cleanup(writer.Clear)
	for _, entry := range []*logger.LogEntry{
		logger.NewLogEntry(logger.INFO, "App", "应用启动成功"),
		logger.NewLogEntry(logger.INFO, "LaunchServer", "LaunchServer 已启动"),
		logger.NewLogEntry(logger.INFO, "Browser", "实例配置持久化成功"),
		logger.NewLogEntry(logger.INFO, "Frontend", "窗口已创建"),
		logger.NewLogEntry(logger.ERROR, "Browser", "浏览器启动失败"),
	} {
		if err := writer.Write(entry); err != nil {
			t.Fatal(err)
		}
	}

	entries := (&App{}).GetAppLogs()
	if len(entries) != 5 {
		t.Fatalf("entries = %#v, want complete stream", entries)
	}
	if entries[0].Message != "应用启动成功" || entries[4].Level != "ERROR" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
