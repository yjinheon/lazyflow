package views

import (
	"strings"
	"testing"

	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

const pySrc = `from airflow.decorators import dag, task

@dag(schedule="@daily", catchup=False)
def etl():
    return {"n": 42}  # comment
`

func TestHighlightUsesThemeSyntaxStyle(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)
	out := HighlightPython(pySrc)

	if out == pySrc {
		t.Fatal("source came back unhighlighted")
	}
	// Colours from the theme's chroma style must actually appear.
	for _, want := range []string{"#7aa2f7", "#bb9af7", "#c0caf5"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected colour %s in markup", want)
		}
	}
	// Every source line must survive.
	for _, line := range []string{"from", "airflow", "@daily", "comment"} {
		if !strings.Contains(out, line) {
			t.Errorf("source text %q lost during highlighting", line)
		}
	}
}

func TestHighlightFallsBackSafely(t *testing.T) {
	theme.ApplyTheme(theme.TokyoNightStorm)

	if got := HighlightPython(""); got != "" {
		t.Errorf("empty input produced %q", got)
	}

	// Oversized input skips the lexer and returns escaped plain text.
	big := strings.Repeat("x = [0]\n", (highlightLimit/8)+10)
	got := HighlightPython(big)
	if strings.Contains(got, "#") {
		t.Error("oversized input should not be highlighted")
	}
	if !strings.Contains(got, "[0[]") {
		t.Error("oversized fallback must still escape bracket sequences")
	}
}

func TestHighlightUnknownStyleFallsBack(t *testing.T) {
	broken := theme.TokyoNightStorm
	broken.SyntaxStyle = "does-not-exist"
	theme.ApplyTheme(broken)
	defer theme.ApplyTheme(theme.TokyoNightStorm)

	// chroma returns its default style for unknown names, so the call must
	// still succeed and preserve the source text.
	out := HighlightPython(pySrc)
	if !strings.Contains(out, "airflow") {
		t.Error("source lost when the style name is unknown")
	}
}

func BenchmarkHighlightPython(b *testing.B) {
	theme.ApplyTheme(theme.TokyoNightStorm)
	src := strings.Repeat(pySrc, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = HighlightPython(src)
	}
}
