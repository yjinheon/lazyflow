package views

import (
	"fmt"
	"time"
)

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}
