package ui

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/tmux"
	tea "github.com/charmbracelet/bubbletea"
)

func typeIntoPalette(o *HelpOverlay, text string) *HelpOverlay {
	for _, r := range text {
		if r == ' ' {
			o, _ = o.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		o, _ = o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return o
}

func TestDeriveTrigger(t *testing.T) {
	cases := map[string]string{
		"r":               "r",
		"n/N":             "n",
		"j / Down":        "j",
		"Ctrl+R":          "ctrl+r",
		"Enter":           "enter",
		"⏎":               "enter",
		"Shift+Left":      "shift+left",
		"+ / K / Shift+↑": "+",
		"<":               "<",

		// Reference-only: ranges, sequences, prose, flags.
		"1-9":            "",
		"a-z":            "",
		"n → w":          "",
		"F → w":          "",
		"--group <name>": "",
		"gg / G":         "",
		"":               "",
	}
	for label, want := range cases {
		if got := deriveTrigger(label); got != want {
			t.Errorf("deriveTrigger(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestKeyMsgForRoundTripsThroughNormalize(t *testing.T) {
	home := NewHome()
	t.Cleanup(func() {
		home.cancel()
		if home.storage != nil {
			_ = home.storage.Close()
		}
	})
	// Every trigger the overlay can produce must synthesize a key message whose
	// String() is what the dispatch switch expects — otherwise a palette row
	// would silently run nothing, or worse, run a different command.
	for _, sec := range NewHelpOverlay().buildSections() {
		for _, item := range sec.items {
			if !item.runnable() {
				continue
			}
			msg, ok := keyMsgFor(item.trigger)
			if !ok {
				t.Errorf("row %q (%s): trigger %q cannot be synthesized", item.key, item.desc, item.trigger)
				continue
			}
			if got := home.normalizeMainKey(msg.String()); got == "" {
				t.Errorf("row %q: synthesized key %q normalizes to nothing", item.key, msg.String())
			}
		}
	}
}

func TestPaletteFiltersAndRuns(t *testing.T) {
	o := NewHelpOverlay()
	o.SetSize(90, 40)
	o.Show()

	o = typeIntoPalette(o, "copy visible")
	view := tmux.StripANSI(o.View())
	if !strings.Contains(view, "Copy visible terminal text") {
		t.Fatalf("multi-term filter did not find the copy-pane row:\n%s", view)
	}
	if item, ok := o.selectedItem(); !ok || item.trigger != "V" {
		t.Errorf("expected the copy-pane row preselected, got %+v (ok=%v)", item, ok)
	}
	if strings.Contains(view, "Restart selected session") {
		t.Errorf("filter should have excluded unrelated rows:\n%s", view)
	}

	// Enter runs the highlighted row and closes.
	item, ok := o.selectedItem()
	if !ok {
		t.Fatal("no selectable row after filtering")
	}
	o, _ = o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if o.IsVisible() {
		t.Error("overlay should close when a row is run")
	}
	if got := o.TakeTrigger(); got != item.trigger {
		t.Errorf("TakeTrigger() = %q, want %q", got, item.trigger)
	}
	if got := o.TakeTrigger(); got != "" {
		t.Errorf("trigger should be consumed once, got %q on the second read", got)
	}
}

func TestPaletteEscClearsFilterBeforeClosing(t *testing.T) {
	o := NewHelpOverlay()
	o.SetSize(90, 40)
	o.Show()
	o = typeIntoPalette(o, "rest")
	o, _ = o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !o.IsVisible() {
		t.Fatal("first esc should clear the filter, not close")
	}
	if o.filter != "" {
		t.Errorf("filter = %q, want empty after esc", o.filter)
	}
	o, _ = o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if o.IsVisible() {
		t.Error("second esc should close")
	}
}

func TestPaletteTypingDoesNotClose(t *testing.T) {
	// j and k used to scroll and every other letter closed the overlay, which
	// is incompatible with typing a filter.
	o := NewHelpOverlay()
	o.SetSize(90, 40)
	o.Show()
	o = typeIntoPalette(o, "jk")
	if !o.IsVisible() {
		t.Fatal("typing must not close the palette")
	}
	if o.filter != "jk" {
		t.Errorf("filter = %q, want %q", o.filter, "jk")
	}
	o, _ = o.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if o.filter != "j" {
		t.Errorf("backspace: filter = %q, want %q", o.filter, "j")
	}
}

func TestPaletteSelectionClampsAndSkipsReferenceRows(t *testing.T) {
	o := NewHelpOverlay()
	o.SetSize(90, 40)
	o.Show()

	// "1-9" is a range, not a key: it must never become the selection.
	o = typeIntoPalette(o, "root group")
	if item, ok := o.selectedItem(); ok {
		t.Errorf("reference-only row became selectable: %+v", item)
	}

	o.filter = ""
	o.selected = 0
	for i := 0; i < 500; i++ {
		o.moveSelection(1)
	}
	item, ok := o.selectedItem()
	if !ok {
		t.Fatal("selection ran off the end of the list")
	}
	if !item.runnable() {
		t.Errorf("selection landed on a non-runnable row: %+v", item)
	}
	for i := 0; i < 500; i++ {
		o.moveSelection(-1)
	}
	if o.selected != 0 {
		t.Errorf("selection should clamp at 0, got %d", o.selected)
	}
}

func TestPaletteRunDispatchesOnTheList(t *testing.T) {
	home := NewHome()
	t.Cleanup(func() {
		home.cancel()
		if home.storage != nil {
			_ = home.storage.Close()
		}
	})
	home.width, home.height = 120, 40
	home.flatItems = []session.Item{{
		Type:    session.ItemTypeSession,
		Session: &session.Instance{ID: "palette", Title: "palette"},
	}}
	home.helpOverlay.SetSize(120, 40)
	home.helpOverlay.Show()

	// Filter to the settings row and run it: the settings panel must open,
	// proving the replayed key reached handleMainKey.
	home.helpOverlay = typeIntoPalette(home.helpOverlay, "settings")
	if _, ok := home.helpOverlay.selectedItem(); !ok {
		t.Fatal("no runnable row for 'settings'")
	}
	model, _ := home.Update(tea.KeyMsg{Type: tea.KeyEnter})
	h, ok := model.(*Home)
	if !ok {
		t.Fatalf("Update returned %T", model)
	}
	if h.helpOverlay.IsVisible() {
		t.Error("palette should have closed")
	}
	if !h.settingsPanel.IsVisible() {
		t.Error("running the settings row should have opened the settings panel")
	}
}

// Document order alone picks the wrong row: "Edit session settings" is printed
// in SESSIONS, long before the "Settings" row in OTHER.
func TestPalettePreselectsTheBestMatch(t *testing.T) {
	cases := []struct {
		query   string
		trigger string
	}{
		{"settings", "S"},
		{"help", "?"},
		{"rename session", "r"},
		{"quit", "q"},
	}
	for _, tc := range cases {
		o := NewHelpOverlay()
		o.SetSize(90, 40)
		o.Show()
		o = typeIntoPalette(o, tc.query)
		item, ok := o.selectedItem()
		if !ok {
			t.Errorf("%q: no selection", tc.query)
			continue
		}
		if item.trigger != tc.trigger {
			t.Errorf("%q preselected %q (%s), want trigger %q",
				tc.query, item.trigger, item.desc, tc.trigger)
		}
	}
}
