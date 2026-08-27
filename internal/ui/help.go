package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// wrapWithHangingIndent wraps text to fit within width, indenting continuation
// lines with the given indent string so wrapped descriptions stay aligned under
// their column instead of bleeding back to column 0.
//
// width is the visible character budget for each line (excluding the indent on
// continuation lines). If width <= 0 the text is returned unchanged.
func wrapWithHangingIndent(text string, width int, indent string) string {
	if text == "" || width <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) <= width {
			current += " " + w
			continue
		}
		lines = append(lines, current)
		current = w
	}
	lines = append(lines, current)
	if len(lines) == 1 {
		return lines[0]
	}
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

// HelpOverlay shows keyboard shortcuts in a modal
type HelpOverlay struct {
	visible      bool
	width        int
	height       int
	scrollOffset int // Current scroll position for small screens
	hotkeys      map[string]string
	// hasAgents gates the Agents row. The feature is opt-in by presence, and
	// this overlay is part of the TUI: a user who has adopted nothing must not
	// find a key here for a surface that does not exist for them.
	hasAgents bool

	// Palette state. filter is what has been typed; selected indexes the
	// runnable rows of the filtered list; pendingTrigger is the key Enter
	// chose, collected by the caller through TakeTrigger.
	filter         string
	selected       int
	pendingTrigger string
}

// SetHasAgents records whether anything has been adopted.
func (h *HelpOverlay) SetHasAgents(has bool) {
	if h == nil {
		return
	}
	h.hasAgents = has
}

// NewHelpOverlay creates a new help overlay
func NewHelpOverlay() *HelpOverlay {
	return &HelpOverlay{hotkeys: resolveHotkeys(nil)}
}

// Show makes the help overlay visible
func (h *HelpOverlay) Show() {
	h.visible = true
	h.scrollOffset = 0
	h.filter = ""
	h.selected = 0
	h.pendingTrigger = ""
}

// Hide hides the help overlay
func (h *HelpOverlay) Hide() {
	h.visible = false
}

// IsVisible returns whether the help overlay is visible
func (h *HelpOverlay) IsVisible() bool {
	return h.visible
}

// SetSize sets the dimensions for centering
func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// SetHotkeys updates displayed hotkeys for dynamic help rendering.
func (h *HelpOverlay) SetHotkeys(bindings map[string]string) {
	h.hotkeys = make(map[string]string, len(bindings))
	for action, key := range bindings {
		h.hotkeys[action] = key
	}
}

func (h *HelpOverlay) key(action, fallback string) string {
	if h.hotkeys == nil {
		return fallback
	}
	if key, ok := h.hotkeys[action]; ok {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (h *HelpOverlay) keyPair(a, b, fallback string) string {
	if h.hotkeys == nil {
		return fallback
	}
	joined := joinHotkeyLabels(actionHotkey(h.hotkeys, a), actionHotkey(h.hotkeys, b))
	if joined != "" {
		return joined
	}
	return ""
}

// Update handles messages for the help overlay.
//
// The overlay is a palette, so printable keys type into the filter rather than
// closing it — j and k are filter text now, and scrolling lives on the arrows,
// page keys, and Ctrl+U/D. Esc clears the filter, then closes.
func (h *HelpOverlay) Update(msg tea.Msg) (*HelpOverlay, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return h, nil
	}

	switch key.String() {
	case "esc":
		if h.filter != "" {
			h.filter = ""
			h.selected = 0
			h.scrollOffset = 0
			return h, nil
		}
		h.Hide()
		return h, nil

	case "enter":
		if item, ok := h.selectedItem(); ok {
			h.pendingTrigger = item.trigger
			h.Hide()
		}
		return h, nil

	case "down", "ctrl+n":
		h.moveSelection(1)
		return h, nil
	case "up", "ctrl+p":
		h.moveSelection(-1)
		return h, nil

	case "pgdown":
		h.scrollOffset += 10
		return h, nil
	case "ctrl+u", "pgup":
		if h.scrollOffset > 10 {
			h.scrollOffset -= 10
		} else {
			h.scrollOffset = 0
		}
		return h, nil
	case "ctrl+d":
		h.scrollOffset += 10
		return h, nil
	case "home":
		h.scrollOffset = 0
		return h, nil
	case "end":
		h.scrollOffset = 9999 // Will be clamped in View()
		return h, nil

	case "backspace":
		if runes := []rune(h.filter); len(runes) > 0 {
			h.filter = string(runes[:len(runes)-1])
			h.reselect()
		}
		return h, nil
	case "ctrl+w":
		h.filter = trimLastWord(h.filter)
		h.reselect()
		return h, nil
	}

	// Printable input types into the filter. Space included: multi-word
	// queries ("copy pane") are how the palette narrows.
	if key.Type == tea.KeyRunes && !key.Alt {
		h.filter += string(key.Runes)
		h.reselect()
		return h, nil
	}
	if key.Type == tea.KeySpace {
		h.filter += " "
		h.reselect()
		return h, nil
	}

	// Anything else (a chord we do not handle) closes, as it always did.
	h.Hide()
	return h, nil
}

// reselect moves the highlight to the row that best answers the current
// filter, and resets the scroll so the answer is on screen.
func (h *HelpOverlay) reselect() {
	h.selected = defaultSelection(filterSections(h.buildSections(), h.filter), h.filter)
	h.scrollOffset = 0
}

// selectedItem returns the highlighted runnable row of the filtered list.
func (h *HelpOverlay) selectedItem() (helpItem, bool) {
	runnable := runnableItems(filterSections(h.buildSections(), h.filter))
	if h.selected < 0 || h.selected >= len(runnable) {
		return helpItem{}, false
	}
	return runnable[h.selected], true
}

// moveSelection walks the runnable rows, clamping at both ends rather than
// wrapping: a palette that jumps from bottom to top loses the reader's place.
func (h *HelpOverlay) moveSelection(delta int) {
	count := len(runnableItems(filterSections(h.buildSections(), h.filter)))
	if count == 0 {
		h.selected = 0
		return
	}
	next := h.selected + delta
	if next < 0 {
		next = 0
	}
	if next >= count {
		next = count - 1
	}
	h.selected = next
}

// trimLastWord drops the trailing word of the filter (Ctrl+W).
func trimLastWord(s string) string {
	trimmed := strings.TrimRight(s, " ")
	if idx := strings.LastIndex(trimmed, " "); idx >= 0 {
		return trimmed[:idx+1]
	}
	return ""
}

// View renders the help overlay
func (h *HelpOverlay) View() string {
	if !h.visible {
		return ""
	}

	sections := h.buildSections()

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent)

	sectionStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)

	// Responsive dialog width: prefer wider so descriptions don't wrap
	// awkwardly. Default 70, scale up to ~80 when the terminal is roomy; the
	// shared helper handles shrinking (and the never-overflow clamp) on narrow
	// terminals.
	preferred := 70
	if h.width >= 100 {
		preferred = 80
	}
	dialogWidth := fitDialogWidth(preferred, 35, h.width)
	keyWidth := 14
	if dialogWidth < 45 {
		keyWidth = 10 // Compact key column for small screens
	}
	// Description column budget: dialogWidth minus border (2) + padding (4)
	// + leading "  " (2) + key column. Hanging indent for wrapped lines is
	// the same width as the leading spaces + key column so continuations sit
	// aligned under the description column.
	descWidth := dialogWidth - 2 - 4 - 2 - keyWidth
	if descWidth < 10 {
		descWidth = 10
	}
	hangingIndent := strings.Repeat(" ", 2+keyWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Width(keyWidth)

	descStyle := lipgloss.NewStyle().
		Foreground(ColorText)

	separatorStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	versionStyle := lipgloss.NewStyle().
		Foreground(ColorComment).
		Italic(true)
	footerStyle := lipgloss.NewStyle().
		Foreground(ColorComment).
		Italic(true)
	scrollIndicatorStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Bold(true)

	// Palette styles: the selection is a filled chip so it reads at a glance,
	// reference-only rows are dimmed so the eye skips them.
	selectedKeyStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Width(keyWidth)
	selectedDescStyle := lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	referenceStyle := lipgloss.NewStyle().Foreground(ColorComment)
	const selectedMarker = "▸ "

	// The filter narrows the list; the selection walks the runnable rows of
	// whatever survives.
	sections = filterSections(sections, h.filter)
	runnable := runnableItems(sections)
	if h.selected >= len(runnable) {
		h.selected = len(runnable) - 1
	}
	if h.selected < 0 {
		h.selected = 0
	}
	runnableIndex := 0
	selectedLine := -1

	// Build content as lines for scrolling support
	var lines []string

	lines = append(lines, titleStyle.Render("KEYBOARD SHORTCUTS"))
	filterLabel := lipgloss.NewStyle().Foreground(ColorComment).Italic(true)
	if h.filter == "" {
		lines = append(lines, filterLabel.Render("type to filter · ↑↓ select · enter run · esc close"))
	} else {
		queryStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
		summary := "no match"
		if len(runnable) > 0 {
			summary = "enter runs the highlighted row"
		}
		lines = append(lines, filterLabel.Render("filter ")+queryStyle.Render(h.filter)+filterLabel.Render("  ·  "+summary))
	}
	lines = append(lines, "")

	if len(sections) == 0 {
		lines = append(lines, referenceStyle.Render("  nothing matches "+h.filter))
	}

	for i, section := range sections {
		lines = append(lines, sectionStyle.Render(section.title))
		for _, item := range section.items {
			wrapped := wrapWithHangingIndent(item.desc, descWidth, hangingIndent)
			marker := "  "
			rowKeyStyle, rowDescStyle := keyStyle, descStyle
			if item.runnable() {
				if runnableIndex == h.selected {
					marker = selectedMarker
					rowKeyStyle = selectedKeyStyle
					rowDescStyle = selectedDescStyle
					selectedLine = len(lines)
				}
				runnableIndex++
			} else {
				// Reference rows (ranges, sequences, startup flags) cannot be
				// run from here, so they are never the selection.
				rowDescStyle = referenceStyle
			}
			lines = append(lines, marker+rowKeyStyle.Render(item.key)+rowDescStyle.Render(wrapped))
		}
		if i < len(sections)-1 {
			lines = append(lines, "")
		}
	}

	// Version info
	separatorWidth := dialogWidth - 8
	if separatorWidth < 20 {
		separatorWidth = 20
	}
	lines = append(lines, "")
	lines = append(lines, separatorStyle.Render(strings.Repeat("─", separatorWidth)))
	lines = append(lines, versionStyle.Render("Agent Deck v"+Version))

	totalLines := len(lines)

	// Calculate available height for content (screen height minus dialog borders, padding, footer)
	// Dialog box has 2 lines for border (top/bottom) + 1 padding each side + 2 for footer area
	availableHeight := h.height - 8
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Check if scrolling is needed
	needsScroll := totalLines > availableHeight

	// Clamp scroll offset
	maxScroll := totalLines - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if h.scrollOffset > maxScroll {
		h.scrollOffset = maxScroll
	}
	if h.scrollOffset < 0 {
		h.scrollOffset = 0
	}

	// Keep the selected row on screen: filtering and arrow-key selection both
	// move it, and a selection scrolled out of view is worse than no selection.
	if selectedLine >= 0 && needsScroll {
		if selectedLine < h.scrollOffset+1 {
			h.scrollOffset = selectedLine - 1
		}
		if selectedLine >= h.scrollOffset+availableHeight-1 {
			h.scrollOffset = selectedLine - availableHeight + 2
		}
		if h.scrollOffset > maxScroll {
			h.scrollOffset = maxScroll
		}
		if h.scrollOffset < 0 {
			h.scrollOffset = 0
		}
	}

	// Build visible content
	var content strings.Builder

	if needsScroll {
		// Show scroll indicator at top if not at beginning
		if h.scrollOffset > 0 {
			content.WriteString(scrollIndicatorStyle.Render("▲ more above"))
			content.WriteString("\n")
			availableHeight-- // Account for indicator line
		}

		// Determine end index
		endIdx := h.scrollOffset + availableHeight
		if h.scrollOffset > 0 {
			// Leave room for bottom indicator if needed
			if endIdx < totalLines {
				availableHeight--
				endIdx = h.scrollOffset + availableHeight
			}
		}
		if endIdx > totalLines {
			endIdx = totalLines
		}

		// Render visible lines
		for i := h.scrollOffset; i < endIdx; i++ {
			content.WriteString(lines[i])
			if i < endIdx-1 {
				content.WriteString("\n")
			}
		}

		// Show scroll indicator at bottom if more content below
		if endIdx < totalLines {
			content.WriteString("\n")
			content.WriteString(scrollIndicatorStyle.Render("▼ more below"))
		}
	} else {
		// No scrolling needed, render all lines
		for i, line := range lines {
			content.WriteString(line)
			if i < len(lines)-1 {
				content.WriteString("\n")
			}
		}
	}

	// Footer hint. j/k are filter text now, so scrolling is advertised on the
	// keys that still scroll.
	content.WriteString("\n\n")
	hint := "↑↓ select • enter run • esc close"
	if needsScroll {
		hint = "↑↓ select • PgUp/PgDn scroll • enter run • esc close"
	}
	if h.filter != "" {
		hint = "esc clears the filter • " + hint
	}
	content.WriteString(footerStyle.Render(hint))

	// Wrap in dialog box
	box := DialogBoxStyle.
		Width(dialogWidth).
		Render(content.String())

	return centerInScreen(box, h.width, h.height)
}
