package api

import "testing"

// The rendered shape is load-bearing: views.HighlightLogs parses "[ts] {logger}
// LEVEL - message" positionally, so dropping a field must not shift the rest.
func TestFormatLogLine(t *testing.T) {
	tests := []struct {
		name                            string
		timestamp, logger, level, event string
		want                            string
	}{
		{
			name:      "all fields",
			timestamp: "2026-08-08T00:12:00.123456+00:00",
			logger:    "task",
			level:     "info",
			event:     "Marking task as SUCCESS",
			want:      "[2026-08-08T00:12:00.123456+00:00] {task} INFO - Marking task as SUCCESS\n",
		},
		{
			name:      "no logger",
			timestamp: "2026-08-08T00:12:00+00:00",
			level:     "error",
			event:     "boom",
			want:      "[2026-08-08T00:12:00+00:00] ERROR - boom\n",
		},
		{
			name:  "event only",
			event: "bare line",
			want:  "bare line\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatLogLine(tt.timestamp, tt.logger, tt.level, tt.event); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
