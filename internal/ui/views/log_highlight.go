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
	body := logTag(th.PrimaryText, "-")

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

	b.WriteString(msg)
	b.WriteString(tview.Escape(rest))
}

// levelStyle returns the label markup and, for severe levels, markup to tint
// the rest of the line with.
func levelStyle(level string) (label, carry string) {
	th := theme.ActiveTheme()
	switch level {
	case "ERROR", "CRITICAL", "FATAL":
		return logTag(th.StatusFailed, "b"), logTag(th.StatusFailed, "-")
	case "WARNING", "WARN":
		return logTag(th.StatusPaused, "b"), logTag(th.StatusPaused, "-")
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
