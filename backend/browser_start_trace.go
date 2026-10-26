package backend

import (
	"ant-chrome/backend/internal/logger"
	"fmt"
	"strings"
	"time"
)

// Keep timings in the message as well as structured fields: the log page shows
// message text, while exported logs retain the original fields for diagnosis.
type browserStartupTrace struct {
	category  string
	profileID string
	started   time.Time
	previous  time.Time
	parts     []string
	fields    map[string]int64
}

func newBrowserStartupTrace(category, profileID string) *browserStartupTrace {
	now := time.Now()
	return &browserStartupTrace{category: category, profileID: profileID, started: now, previous: now, fields: map[string]int64{}}
}

func (t *browserStartupTrace) step(name string) {
	now := time.Now()
	ms := now.Sub(t.previous).Milliseconds()
	t.previous = now
	t.fields[name] += ms
	t.parts = append(t.parts, fmt.Sprintf("%s=%d ms", name, ms))
}

func (t *browserStartupTrace) finish() {
	t.step("返回前剩余")
	id := t.profileID
	if len(id) > 8 {
		id = id[:8]
	}
	total := time.Since(t.started).Milliseconds()
	logger.New("Browser").Info(fmt.Sprintf("%s [%s]：总计 %d ms；%s", t.category, id, total, strings.Join(t.parts, "；")),
		logger.F("profile_id", t.profileID), logger.F("total_ms", total), logger.F("phases_ms", t.fields))
}
