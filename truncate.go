package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// truncateDisplayWidth truncates a string to fit within maxWidth display cells.
// It is ANSI-escape aware and runewidth-aware.
func truncateDisplayWidth(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	var b strings.Builder

	w := 0
	inEsc := false

	for _, r := range s {
		if inEsc {
			b.WriteRune(r)

			if r == 'm' {
				inEsc = false
			}

			continue
		}

		if r == '\x1b' {
			inEsc = true

			b.WriteRune(r)

			continue
		}

		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth {
			break
		}

		w += rw

		b.WriteRune(r)
	}

	return b.String()
}
