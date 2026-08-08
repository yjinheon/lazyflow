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

// A bracket that is not Rich markup must reach tview escaped, never swallowed
// as a colour tag.
func TestHighlightLogs_escapesNonMarkupBrackets(t *testing.T) {
	tests := map[string]string{
		"{a.py:1} INFO - got [not a tag] from upstream": "[not a tag[]",
		"{a.py:1} INFO - shape [1024] rows":             "[1024[]",
		"{a.py:1} INFO - key [INFO] missing":            "[INFO[]",
	}
	for in, want := range tests {
		if out := HighlightLogs(in); !strings.Contains(out, want) {
			t.Errorf("bracket not escaped in %q:\n got %q\nwant substring %q", in, out, want)
		}
	}
}

// Rich markup printed by DAG code becomes real colour instead of literal tags.
func TestHighlightLogs_convertsRichMarkup(t *testing.T) {
	body := theme.MarkupHex(theme.ActiveTheme().PrimaryText)

	cyan := theme.MarkupHex(theme.ActiveTheme().SectionHeader)
	green := theme.MarkupHex(theme.ActiveTheme().StatusSuccess)

	out := HighlightLogs("{task.stdout} INFO - [cyan]transform_b[/cyan]: rows=[green]4[/green]")
	for _, want := range []string{"[" + cyan + "::-]transform_b", "[" + green + "::-]4"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	// Closing tags restore the message colour rather than leaking the last one.
	if !strings.Contains(out, "["+green+"::-]4["+body+"::-]") {
		t.Errorf("close tag did not restore body colour: %q", out)
	}
	if strings.Contains(out, "[/cyan") || strings.Contains(out, "[/green") {
		t.Errorf("closing tag left in output: %q", out)
	}
}

func TestHighlightLogs_richNesting(t *testing.T) {
	body := theme.MarkupHex(theme.ActiveTheme().PrimaryText)
	out := HighlightLogs("INFO - [bold][red]bad[/red] still bold[/]done")
	for _, want := range []string{
		"[" + theme.MarkupHex(theme.ActiveTheme().StatusFailed) + "::b]bad", // folded onto the enclosing bold
		"[" + body + "::b] still", // [/red] keeps bold, restores body colour
		"[" + body + "::-]done",   // [/] drops bold too
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
}

func TestHighlightLogs_richBrightAndHex(t *testing.T) {
	out := HighlightLogs("INFO - [bright_red]hot[/] [#ff8800]warm[/]")
	if !strings.Contains(out, "["+theme.MarkupHex(theme.ActiveTheme().StatusFailed)+"::b]hot") {
		t.Errorf("bright_red should render as bold red: %q", out)
	}
	if !strings.Contains(out, "[#ff8800::-]warm") {
		t.Errorf("hex colour not applied: %q", out)
	}
}

// An unmatched close must not pop past the message style.
func TestHighlightLogs_richUnbalancedClose(t *testing.T) {
	body := theme.MarkupHex(theme.ActiveTheme().PrimaryText)
	out := HighlightLogs("INFO - [/cyan]plain")
	if !strings.Contains(out, "["+body+"::-]plain") {
		t.Errorf("stray close broke the body style: %q", out)
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
