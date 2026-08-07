package views

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// stripTags removes tview colour tags so the visible text can be compared.
var tagRe = regexp.MustCompile(`\[[^\]]*\]`)

func TestHighlightLogs_preservesVisibleText(t *testing.T) {
	src := "[2026-08-08T00:12:00.123+0900] {taskinstance.py:1234} INFO - Marking task as SUCCESS\n" +
		"[2026-08-08T00:12:01.000+0900] {ti.py:9} ERROR - boom\n" +
		"Traceback (most recent call last):\n" +
		`  File "/dags/etl.py", line 3, in <module>`

	got := tagRe.ReplaceAllString(HighlightLogs(src), "")
	want := tagRe.ReplaceAllString(src, "")
	if got != want {
		t.Errorf("visible text changed:\n got %q\nwant %q", got, want)
	}
}

// A bracket in the message must reach tview escaped, not read as a colour tag.
func TestHighlightLogs_escapesBrackets(t *testing.T) {
	out := HighlightLogs("{a.py:1} INFO - got [red] from upstream")
	if !strings.Contains(out, "[red[]") {
		t.Errorf("bracket not escaped: %q", out)
	}
}

func TestHighlightLogs_colorsBySeverity(t *testing.T) {
	th := theme.ActiveTheme()
	red := theme.MarkupHex(th.StatusFailed)
	yellow := theme.MarkupHex(th.StatusPaused)

	out := HighlightLogs("[2026-01-01T00:00:00+0000] {a.py:1} ERROR - boom")
	if !strings.Contains(out, red) {
		t.Errorf("ERROR line missing %s: %q", red, out)
	}

	out = HighlightLogs("[2026-01-01T00:00:00+0000] {a.py:1} WARNING - careful")
	if !strings.Contains(out, yellow) {
		t.Errorf("WARNING line missing %s: %q", yellow, out)
	}

	out = HighlightLogs("[2026-01-01T00:00:00+0000] {a.py:1} INFO - fine")
	if strings.Contains(out, red) {
		t.Errorf("INFO line should not be red: %q", out)
	}
}

// Guards the contract with internal/api.formatLogLine: every part of the line
// shape it emits has to pick up a colour.
func TestHighlightLogs_apiLineShape(t *testing.T) {
	th := theme.ActiveTheme()
	out := HighlightLogs("[2026-08-08T00:12:00.123456+00:00] {task} INFO - Marking task as SUCCESS")

	for name, want := range map[string]string{
		"timestamp": theme.MarkupHex(th.MutedText),
		"logger":    theme.MarkupHex(th.SectionHeader),
		"level":     theme.MarkupHex(th.Accent),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%s not coloured (%s missing): %q", name, want, out)
		}
	}
}

func TestHighlightLogs_plainLineIsUntagged(t *testing.T) {
	out := HighlightLogs("just a bare line")
	if got := tagRe.ReplaceAllString(out, ""); got != "just a bare line" {
		t.Errorf("got %q", got)
	}
}

func TestHighlightLogs_empty(t *testing.T) {
	if got := HighlightLogs(""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
