package views

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// highlightLimit caps the input; tokenising costs ~60ms per 1k lines, so past
// this size the plain text is served instead.
const highlightLimit = 512 << 10

// HighlightPython renders Python source as tview markup using the active
// theme's chroma style. It is CPU-bound — call it off the tview goroutine.
// Any failure falls back to escaped plain text.
func HighlightPython(src string) string {
	if src == "" || len(src) > highlightLimit {
		return tview.Escape(src)
	}
	lexer := lexers.Get("python")
	style := styles.Get(theme.ActiveTheme().SyntaxStyle)
	if lexer == nil || style == nil {
		return tview.Escape(src)
	}
	it, err := lexer.Tokenise(nil, src)
	if err != nil {
		return tview.Escape(src)
	}
	return tokensToMarkup(it.Tokens(), style)
}

// tokensToMarkup groups adjacent tokens that share a style into one run and
// escapes each run as a whole. Escaping per token would not work: the lexer
// splits "[red]" into three tokens, and tview only escapes a bracket sequence
// it can see in full — otherwise it swallows it as a colour tag.
func tokensToMarkup(tokens []chroma.Token, style *chroma.Style) string {
	fallback := theme.MarkupHex(theme.ActiveTheme().PrimaryText)

	var b, run strings.Builder
	b.Grow(len(tokens) * 8)
	prev := ""

	flush := func() {
		if run.Len() > 0 {
			b.WriteString(tview.Escape(run.String()))
			run.Reset()
		}
	}
	for _, tok := range tokens {
		if tok.Value == "" {
			continue
		}
		if tag := styleTag(style.Get(tok.Type), fallback); tag != prev {
			flush()
			b.WriteString(tag)
			prev = tag
		}
		run.WriteString(tok.Value)
	}
	flush()
	return b.String()
}

// styleTag builds a tview "[fg::flags]" tag for one chroma style entry.
func styleTag(e chroma.StyleEntry, fallback string) string {
	colour := fallback
	if e.Colour.IsSet() {
		colour = e.Colour.String()
	}
	flags := ""
	if e.Bold == chroma.Yes {
		flags += "b"
	}
	if e.Italic == chroma.Yes {
		flags += "i"
	}
	if flags == "" {
		flags = "-"
	}
	return "[" + colour + "::" + flags + "]"
}
