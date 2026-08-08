package views

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
)

// Airflow log lines are regular: "[ts] {file.py:12} INFO - message".
var (
	logTimestampRe = regexp.MustCompile(`^\[\d[^\]]*\]`)
	logSourceRe    = regexp.MustCompile(`^\s*\{[^}]*\}`)
	logLevelRe     = regexp.MustCompile(`(?i)^\s*(DEBUG|INFO|WARNING|WARN|ERROR|CRITICAL|FATAL)\b`)
	logTracebackRe = regexp.MustCompile(`^(Traceback \(most recent call last\)|\s+File ")`)
)

// HighlightLogs renders task logs as tview markup. CPU-bound like
// HighlightPython — call it off the tview goroutine.
func HighlightLogs(text string) string {
	if text == "" || len(text) > highlightLimit {
		return tview.Escape(text)
	}
	th := theme.ActiveTheme()
	muted := logTag(th.MutedText, "-")
	source := logTag(th.SectionHeader, "-")
	body := theme.MarkupHex(th.PrimaryText)

	lines := strings.Split(text, "\n")
	var b strings.Builder
	b.Grow(len(text) + len(lines)*24)
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeLogLine(&b, line, muted, source, body)
	}
	return b.String()
}

func writeLogLine(b *strings.Builder, line, muted, source, body string) {
	// Traceback continuation lines carry no level of their own.
	if logTracebackRe.MatchString(line) {
		label, _ := levelStyle("ERROR")
		b.WriteString(label)
		b.WriteString(tview.Escape(line))
		return
	}

	rest := line
	if m := logTimestampRe.FindString(rest); m != "" {
		b.WriteString(muted)
		b.WriteString(tview.Escape(m))
		rest = rest[len(m):]
	}
	if m := logSourceRe.FindString(rest); m != "" {
		b.WriteString(source)
		b.WriteString(tview.Escape(m))
		rest = rest[len(m):]
	}

	msg := body
	if m := logLevelRe.FindString(rest); m != "" {
		label, carry := levelStyle(strings.ToUpper(strings.TrimSpace(m)))
		b.WriteString(label)
		b.WriteString(tview.Escape(m))
		rest = rest[len(m):]
		if carry != "" {
			msg = carry
		}
	}

	writeRichBody(b, rest, richStyle{colour: msg})
}

// levelStyle returns the label markup and, for severe levels, the colour to
// tint the rest of the line with.
func levelStyle(level string) (label, carryColour string) {
	th := theme.ActiveTheme()
	switch level {
	case "ERROR", "CRITICAL", "FATAL":
		return logTag(th.StatusFailed, "b"), theme.MarkupHex(th.StatusFailed)
	case "WARNING", "WARN":
		return logTag(th.StatusPaused, "b"), theme.MarkupHex(th.StatusPaused)
	case "DEBUG":
		return logTag(th.MutedText, "-"), ""
	default:
		return logTag(th.Accent, "-"), ""
	}
}

// logTag spells out flags so a tag never inherits bold from the previous one.
func logTag(c tcell.Color, flags string) string {
	return "[" + theme.MarkupHex(c) + "::" + flags + "]"
}

// ---------- Rich console markup ----------
//
// DAG code that prints through Rich (or Typer/Click) emits "[cyan]x[/cyan]"
// into the log stream. Escaping it wholesale shows the tags as text, so known
// tags are translated to tview styles instead; anything unrecognised is still
// escaped so a stray bracket can never be swallowed as a colour tag.

// richTagRe matches "[/]", "[/cyan]", "[cyan]", "[bold red]", "[#ff8800]".
var richTagRe = regexp.MustCompile(`\[/\]|\[/?[a-zA-Z#][a-zA-Z0-9_# ]*\]`)

// richColour resolves an ANSI colour name onto the theme palette so log output
// matches the rest of the UI. Names are not passed through to tview: tcell
// resolves W3C names only, and would silently render "cyan"/"magenta" as the
// default colour.
func richColour(name string) (string, bool) {
	th := theme.ActiveTheme()
	switch name {
	case "black":
		return theme.MarkupHex(th.SecondaryBg), true
	case "red":
		return theme.MarkupHex(th.StatusFailed), true
	case "green":
		return theme.MarkupHex(th.StatusSuccess), true
	case "yellow":
		return theme.MarkupHex(th.StatusPaused), true
	case "blue":
		return theme.MarkupHex(th.Accent), true
	case "magenta":
		return theme.MarkupHex(th.StatusQueued), true
	case "cyan":
		return theme.MarkupHex(th.SectionHeader), true
	case "white":
		return theme.MarkupHex(th.PrimaryText), true
	case "grey", "gray":
		return theme.MarkupHex(th.MutedText), true
	}
	return "", false
}

type richStyle struct {
	colour string // tview colour token
	flags  string // accumulated tview flag letters
}

func (s richStyle) tag() string {
	flags := s.flags
	if flags == "" {
		flags = "-"
	}
	colour := s.colour
	if colour == "" {
		colour = "-"
	}
	return "[" + colour + "::" + flags + "]"
}

// writeRichBody emits text with Rich tags converted, starting from base.
func writeRichBody(b *strings.Builder, text string, base richStyle) {
	stack := []richStyle{base}
	b.WriteString(base.tag())

	for {
		loc := richTagRe.FindStringIndex(text)
		if loc == nil {
			break
		}
		b.WriteString(tview.Escape(text[:loc[0]]))
		tag := text[loc[0]+1 : loc[1]-1]
		text = text[loc[1]:]

		switch {
		case strings.HasPrefix(tag, "/"):
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
			b.WriteString(stack[len(stack)-1].tag())
		default:
			s, ok := parseRichTag(tag, stack[len(stack)-1])
			if !ok {
				b.WriteString(tview.Escape("[" + tag + "]"))
				continue
			}
			stack = append(stack, s)
			b.WriteString(s.tag())
		}
	}
	b.WriteString(tview.Escape(text))
}

// parseRichTag folds one tag's words onto the enclosing style. It reports false
// for anything it does not fully understand, so the text stays literal.
func parseRichTag(tag string, parent richStyle) (richStyle, bool) {
	words := strings.Fields(tag)
	if len(words) == 0 {
		return richStyle{}, false
	}
	s := parent
	for _, w := range words {
		switch w {
		case "bold":
			s.flags = addFlag(s.flags, 'b')
		case "dim":
			s.flags = addFlag(s.flags, 'd')
		case "italic":
			s.flags = addFlag(s.flags, 'i')
		case "underline":
			s.flags = addFlag(s.flags, 'u')
		case "strike":
			s.flags = addFlag(s.flags, 's')
		default:
			name := strings.TrimPrefix(w, "bright_")
			if name != w {
				s.flags = addFlag(s.flags, 'b')
			}
			colour, ok := richColour(name)
			if !ok {
				if !strings.HasPrefix(w, "#") || len(w) != 7 {
					return richStyle{}, false
				}
				colour = w
			}
			s.colour = colour
		}
	}
	return s, true
}

func addFlag(flags string, f byte) string {
	if strings.IndexByte(flags, f) >= 0 {
		return flags
	}
	return flags + string(f)
}
