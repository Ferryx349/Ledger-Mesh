package reconcile

import (
	"testing"
	"time"
)

func TestDefaultPeriod(t *testing.T) {
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	start, end := DefaultPeriod(now)
	if !end.Equal(now) {
		t.Fatalf("end=%v", end)
	}
	if !start.Equal(now.Add(-24 * time.Hour)) {
		t.Fatalf("start=%v", start)
	}
}
