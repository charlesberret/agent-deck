package ui

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/tmux"
	"github.com/charmbracelet/lipgloss"
)

func TestFitFooterRowKeepsHelpOverContext(t *testing.T) {
	context := []string{"one-hint", "two-hint", "three-hint", "four-hint"}
	globals := []footerGlobal{
		{"nav", footerDropNav},
		{"search", footerDropSearch},
		{"quit", footerDropQuit},
		{"settings", footerDropSettings},
		{"help", footerDropHelp},
	}

	// Roomy: everything renders.
	row := fitFooterRow(120, "Session: ", context, globals, " | ", 2)
	for _, want := range []string{"one-hint", "four-hint", "help", "settings"} {
		if !strings.Contains(row, want) {
			t.Errorf("wide row missing %q: %q", want, row)
		}
	}

	// Tight: context sheds first, globals survive.
	row = fitFooterRow(40, "Session: ", context, globals, " | ", 2)
	if !strings.Contains(row, "help") {
		t.Errorf("help dropped at width 40: %q", row)
	}
	if strings.Contains(row, "four-hint") {
		t.Errorf("lowest-priority context hint should have been dropped: %q", row)
	}

	// Cramped: globals shed by tier, help last.
	row = fitFooterRow(18, "Session: ", context, globals, " | ", 2)
	if !strings.Contains(row, "help") {
		t.Errorf("help must survive every trim: %q", row)
	}
	for _, gone := range []string{"nav", "search", "quit"} {
		if strings.Contains(row, gone) {
			t.Errorf("%q should have been shed before help: %q", gone, row)
		}
	}
}

func TestFitFooterRowNeverOverflowsWhenItCanFit(t *testing.T) {
	globals := []footerGlobal{{"settings", footerDropSettings}, {"help", footerDropHelp}}
	for _, width := range []int{101, 110, 140, 190} {
		row := fitFooterRow(width, "Session: ", []string{"a", "bb", "ccc", "dddd"}, globals, " | ", 2)
		if w := lipgloss.Width(row); w > width {
			t.Errorf("width %d: row is %d cols: %q", width, w, row)
		}
	}
}

// The full footer is the default on wide terminals. It used to drop the whole
// right-hand block once the context hints filled the row, which is the normal
// case with a session selected — so the help and settings keys were never on
// screen and there was no way to learn them from the UI.
func TestFullFooterAlwaysAdvertisesHelpAndSettings(t *testing.T) {
	home := NewHome()
	t.Cleanup(func() {
		home.cancel()
		if home.storage != nil {
			_ = home.storage.Close()
		}
	})
	home.height = 40
	home.flatItems = []session.Item{{
		Type:    session.ItemTypeSession,
		Session: &session.Instance{ID: "footer-probe", Title: "agent-deck-admin"},
	}}

	helpKey := home.actionKey(hotkeyHelp)
	settingsKey := home.actionKey(hotkeySettings)
	for _, width := range []int{101, 110, 140, 190, 240} {
		home.width = width
		footer := tmux.StripANSI(home.renderHelpBarFull())
		if !strings.Contains(footer, helpKey+" Help") {
			t.Errorf("width %d: footer does not advertise help:\n%s", width, footer)
		}
		if !strings.Contains(footer, settingsKey+" Settings") {
			t.Errorf("width %d: footer does not advertise settings:\n%s", width, footer)
		}
		for _, line := range strings.Split(footer, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: footer line overflows at %d cols:\n%s", width, w, line)
			}
		}
	}
}
