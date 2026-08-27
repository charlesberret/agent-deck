package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// joinPillsFlow lays out pre-rendered pill strings into rows that fit maxWidth
// display cells, wrapping only between pills (never mid-pill).
//
// Subsequent rows are indented with two spaces so they line up under the
// caller's leading indent when the first row is written after "  ".
//
// Without this, a single JoinHorizontal line longer than the dialog Width is
// reflowed by lipgloss at arbitrary cell boundaries, which splits tool names
// across lines and looks broken.
func joinPillsFlow(pills []string, maxWidth int) string {
	if len(pills) == 0 {
		return ""
	}
	if maxWidth < 8 {
		maxWidth = 8
	}

	var rows []string
	var row []string
	rowW := 0

	for _, p := range pills {
		if p == "" {
			continue
		}
		pw := lipgloss.Width(p)
		// If a single pill is wider than the budget, still emit it alone
		// (better one long pill than infinite empty rows).
		if len(row) > 0 && rowW+pw > maxWidth {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, row...))
			row = []string{p}
			rowW = pw
			continue
		}
		row = append(row, p)
		rowW += pw
	}
	if len(row) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, row...))
	}
	if len(rows) == 0 {
		return ""
	}
	// Indent continuation rows to match the "  " prefix callers put before the first row.
	var b strings.Builder
	b.WriteString(rows[0])
	for _, r := range rows[1:] {
		b.WriteString("\n  ")
		b.WriteString(r)
	}
	return b.String()
}

// pillsInnerWidth is the usable cell budget for tool pills inside a dialog
// with Padding(2, 4) and a leading "  " content indent.
func pillsInnerWidth(dialogWidth int) int {
	// border is outside Width; Padding(2,4) consumes 8 cols of content.
	// Leading "  " before the pill row is another 2, but joinPillsFlow's
	// first row is written after that indent — so budget = dialogWidth - 8.
	w := dialogWidth - 8
	if w < 20 {
		w = 20
	}
	return w
}

// radioGroupFlow renders a radio group that wraps between options instead of
// overflowing the dialog. Same marker grammar as SettingsPanel.renderRadioGroup
// (">" on the selection, accent + bold); the two trailing spaces that separate
// options are baked into each entry so joinPillsFlow can break on them.
func radioGroupFlow(options []string, selected, maxWidth int) string {
	entries := make([]string, 0, len(options))
	for i, opt := range options {
		style := lipgloss.NewStyle().Foreground(ColorTextDim)
		marker := " "
		if i == selected {
			style = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
			marker = ">"
		}
		entries = append(entries, style.Render(marker+opt)+"  ")
	}
	// Continuation rows are re-indented to match the caller's leading "  ";
	// joinPillsFlow only knows about the first row's offset.
	return strings.ReplaceAll(joinPillsFlow(entries, maxWidth), "\n", "\n  ")
}
