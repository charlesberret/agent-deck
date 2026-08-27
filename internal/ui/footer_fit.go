package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Drop priorities for the full footer's right-hand global block. Lower is
// dropped sooner; the help key is never dropped, because it is the only thing
// on screen that leads to everything not on screen.
const (
	footerDropNav      = 0 // ↑↓ Nav, +/- Move — the keys people try first anyway
	footerDropSearch   = 1 // search, global search
	footerDropQuit     = 2 // q — discoverable, and Ctrl+C works regardless
	footerDropSettings = 3
	footerDropHelp     = 4
)

// footerGlobal is one right-hand global hint plus how readily it may be
// dropped when the bar is too narrow.
type footerGlobal struct {
	text     string
	priority int
}

// fitFooterRow assembles the full footer's content line.
//
// Context hints (left) are dropped from the lowest-priority end: they describe
// the selected row, which is itself on screen. The global block (right) is
// dropped by priority, so the settings and help keys are the last to go.
//
// Before this, the whole global block was dropped the moment the context hints
// filled the row -- and MaxWidth then clipped the context hints mid-word. On a
// wide terminal with a session selected that is the normal case, so the help
// key was hidden exactly when the interface was busiest, leaving no way to
// discover it from the UI at all.
func fitFooterRow(width int, left string, context []string, globals []footerGlobal, sep string, minGap int) string {
	if minGap < 1 {
		minGap = 1
	}
	ctx := append([]string(nil), context...)
	glb := append([]footerGlobal(nil), globals...)

	renderGlobals := func(items []footerGlobal) string {
		parts := make([]string, 0, len(items))
		for _, g := range items {
			parts = append(parts, g.text)
		}
		return strings.Join(parts, sep)
	}

	fits := func() bool {
		w := lipgloss.Width(left) +
			lipgloss.Width(strings.Join(ctx, " ")) +
			lipgloss.Width(renderGlobals(glb))
		return w+minGap <= width
	}

	shedTier := func(tier int) {
		kept := glb[:0]
		for _, g := range glb {
			if g.priority > tier {
				kept = append(kept, g)
			}
		}
		glb = kept
	}

	// Order of sacrifice, and the reasoning behind it:
	//
	//  1. Globals that are merely convenient — nav, search, quit. Arrow keys
	//     and Ctrl+C are found without a hint, so these buy room cheaply.
	//  2. Context hints, from the lowest-priority end. They describe the
	//     selected row and the help overlay lists them all.
	//  3. Settings, if it still does not fit.
	//
	// Help is never shed. Everything in a tier goes together so the bar never
	// keeps "Move" while dropping "Nav".
	for tier := footerDropNav; tier <= footerDropQuit && !fits(); tier++ {
		shedTier(tier)
	}
	for !fits() && len(ctx) > 0 {
		ctx = ctx[:len(ctx)-1]
	}
	for tier := footerDropQuit + 1; tier < footerDropHelp && !fits(); tier++ {
		shedTier(tier)
	}

	leftPart := left + strings.Join(ctx, " ")
	rightPart := renderGlobals(glb)
	pad := width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if pad < minGap {
		pad = minGap
	}
	return leftPart + strings.Repeat(" ", pad) + rightPart
}
