package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// The help overlay doubles as a command palette: type to filter the rows, move
// with the arrows, press Enter to run the highlighted one. Running a row means
// replaying its key into the main list rather than calling an action directly
// -- the dispatch stays the single switch in handleMainKey, so the palette adds
// no second place where a command can be defined and drift.

// namedTriggers are the multi-character key labels the main list can receive.
// A label outside this set (and longer than one rune) is reference-only: "1-9",
// "a-z", "n → w", "--group <name>" name a range or a sequence, not one key.
var namedTriggers = map[string]string{
	"enter":       "enter",
	"shift+enter": "shift+enter",
	"tab":         "tab",
	"shift+tab":   "shift+tab",
	"esc":         "esc",
	"space":       " ",
	"up":          "up",
	"down":        "down",
	"left":        "left",
	"right":       "right",
	"shift+left":  "shift+left",
	"shift+right": "shift+right",
	"shift+up":    "shift+up",
	"shift+down":  "shift+down",
	"⏎":           "enter",
}

// deriveTrigger reads a help row's key label and returns the key to replay, or
// "" when the label does not name a single receivable key.
//
// Labels come in a few shapes: a bare key ("r"), alternatives ("n/N", "j / Down",
// "+ / K / Shift+↑"), modifiers ("Ctrl+R"), and prose that only looks like a key
// ("1-9", "gg / G", "n → w"). Only the first alternative is offered, and only
// when it is a single rune or a known named key.
func deriveTrigger(label string) string {
	first := strings.TrimSpace(label)
	if first == "" {
		return ""
	}
	// Sequences and ranges are descriptions, not keys.
	if strings.ContainsAny(first, "→<>") && !isSingleRune(first) {
		return ""
	}
	for _, sep := range []string{" / ", "/"} {
		if idx := strings.Index(first, sep); idx > 0 {
			first = strings.TrimSpace(first[:idx])
			break
		}
	}
	if first == "" {
		return ""
	}
	lower := strings.ToLower(first)
	if trigger, ok := namedTriggers[lower]; ok {
		return trigger
	}
	if strings.HasPrefix(lower, "ctrl+") || strings.HasPrefix(lower, "alt+") {
		// Only single-rune chords: "ctrl+r", not "ctrl+shift+something".
		if rest := lower[strings.Index(lower, "+")+1:]; isSingleRune(rest) {
			return lower
		}
		return ""
	}
	if isSingleRune(first) {
		return first
	}
	return ""
}

func isSingleRune(s string) bool {
	return len([]rune(s)) == 1
}

// visibleRows returns the sections filtered by the current query, with empty
// sections dropped. An empty query returns everything.
func filterSections(sections []helpSection, query string) []helpSection {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return sections
	}
	out := make([]helpSection, 0, len(sections))
	for _, sec := range sections {
		kept := helpSection{title: sec.title}
		for _, item := range sec.items {
			haystack := strings.ToLower(item.key + " " + item.desc + " " + sec.title)
			if matchesQuery(haystack, q) {
				kept.items = append(kept.items, item)
			}
		}
		if len(kept.items) > 0 {
			out = append(out, kept)
		}
	}
	return out
}

// matchesQuery requires every whitespace-separated term to appear, so "copy
// pane" narrows rather than widening the way a raw substring match would.
func matchesQuery(haystack, query string) bool {
	for _, term := range strings.Fields(query) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

// runnableItems flattens the filtered sections to the rows Enter can act on,
// in display order.
func runnableItems(sections []helpSection) []helpItem {
	var out []helpItem
	for _, sec := range sections {
		for _, item := range sec.items {
			if item.runnable() {
				out = append(out, item)
			}
		}
	}
	return out
}

// TakeTrigger returns the key the user chose with Enter and clears it, so the
// caller can replay it into the main list exactly once.
func (h *HelpOverlay) TakeTrigger() string {
	if h == nil {
		return ""
	}
	trigger := h.pendingTrigger
	h.pendingTrigger = ""
	return trigger
}

// keyMsgFor builds the tea.KeyMsg for a trigger produced by deriveTrigger.
// Returns ok=false for anything it cannot synthesize, so a row is never
// "run" as some other key.
func keyMsgFor(trigger string) (tea.KeyMsg, bool) {
	switch trigger {
	case "":
		return tea.KeyMsg{}, false
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}, true
	case "shift+enter":
		// Relayed as the private-use rune the csiu reader emits; normalizeMainKey
		// rewrites it back to "shift+enter" before dispatch.
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{shiftEnterMarker}}, true
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}, true
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}, true
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}, true
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, true
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case "shift+left":
		return tea.KeyMsg{Type: tea.KeyShiftLeft}, true
	case "shift+right":
		return tea.KeyMsg{Type: tea.KeyShiftRight}, true
	case "shift+up":
		return tea.KeyMsg{Type: tea.KeyShiftUp}, true
	case "shift+down":
		return tea.KeyMsg{Type: tea.KeyShiftDown}, true
	}
	if strings.HasPrefix(trigger, "ctrl+") || strings.HasPrefix(trigger, "alt+") {
		// bubbletea parses these labels back to the same string it emits, and
		// normalizeMainKey works on msg.String(), so a synthetic rune message
		// carrying the label is not enough — use the typed constants we know.
		if key, ok := chordKeyTypes[trigger]; ok {
			return tea.KeyMsg{Type: key}, true
		}
		if strings.HasPrefix(trigger, "alt+") && isSingleRune(trigger[len("alt+"):]) {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(trigger[len("alt+"):]), Alt: true}, true
		}
		return tea.KeyMsg{}, false
	}
	if isSingleRune(trigger) {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(trigger)}, true
	}
	return tea.KeyMsg{}, false
}

// chordKeyTypes covers the Ctrl chords the help overlay advertises.
var chordKeyTypes = map[string]tea.KeyType{
	"ctrl+r": tea.KeyCtrlR,
	"ctrl+q": tea.KeyCtrlQ,
	"ctrl+z": tea.KeyCtrlZ,
	"ctrl+s": tea.KeyCtrlS,
	"ctrl+u": tea.KeyCtrlU,
	"ctrl+d": tea.KeyCtrlD,
	"ctrl+f": tea.KeyCtrlF,
	"ctrl+b": tea.KeyCtrlB,
	"ctrl+c": tea.KeyCtrlC,
	"ctrl+n": tea.KeyCtrlN,
	"ctrl+p": tea.KeyCtrlP,
}

// defaultSelection picks which runnable row is highlighted after the filter
// changes. Document order alone is a poor answer: typing "settings" would land
// on "Edit session settings" simply because the SESSIONS section is printed
// before OTHER. Rows are scored, and the best-scoring one wins; ties keep
// document order, so the list itself never reshuffles under the reader.
func defaultSelection(sections []helpSection, query string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0
	}
	best, bestScore := 0, -1
	for i, item := range runnableItems(sections) {
		if score := matchScore(item, q); score > bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

// matchScore ranks how squarely a row answers the query. Higher is better.
func matchScore(item helpItem, query string) int {
	desc := strings.ToLower(item.desc)
	key := strings.ToLower(item.key)
	switch {
	case desc == query || key == query:
		return 4
	case strings.HasPrefix(desc, query):
		return 3
	case strings.HasPrefix(key, query):
		return 2
	case strings.Contains(desc, query):
		return 1
	default:
		return 0
	}
}
