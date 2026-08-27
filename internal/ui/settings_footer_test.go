package ui

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/tmux"
)

func TestSettingsPanelFooterRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	session.ClearUserConfigCache()
	t.Cleanup(session.ClearUserConfigCache)

	panel := NewSettingsPanel()
	panel.LoadConfig(&session.UserConfig{UI: session.UISettings{Footer: session.FooterCurated}})
	if got := footerValues[panel.selectedFooter]; got != session.FooterCurated {
		t.Errorf("loaded footer = %q, want curated", got)
	}
	if got := panel.GetConfig().UI.Footer; got != session.FooterCurated {
		t.Errorf("saved footer = %q, want curated", got)
	}

	// An unset footer must land on the default rather than an arbitrary index.
	panel.LoadConfig(&session.UserConfig{})
	if got := footerValues[panel.selectedFooter]; got != (session.UISettings{}).GetFooter() {
		t.Errorf("unset footer loaded as %q, want the default", got)
	}

	// Every mode needs a description: the value names alone do not explain
	// what the setting does, which is why it stayed config-only for so long.
	for _, v := range footerValues {
		if strings.TrimSpace(footerDescs[v]) == "" {
			t.Errorf("footer mode %q has no description", v)
		}
	}
}

func TestSettingsPanelRendersFooterSection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	session.ClearUserConfigCache()
	t.Cleanup(session.ClearUserConfigCache)

	panel := NewSettingsPanel()
	panel.SetSize(120, 60)
	panel.Show()
	panel.cursor = int(SettingFooter)
	view := tmux.StripANSI(panel.View())

	for _, want := range []string{"FOOTER", "Curated", "Minimal", "always kept"} {
		if !strings.Contains(view, want) {
			t.Errorf("settings view missing %q:\n%s", want, view)
		}
	}
}

// The line map used for cursor-anchored scrolling is derived, not hand-counted.
// Its length must track settingsCount, and it must stay monotonic — a setting
// that maps above the one before it scrolls the panel backwards.
func TestSettingsPanelLineMapCoversEverySetting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	session.ClearUserConfigCache()
	t.Cleanup(session.ClearUserConfigCache)

	for _, themeValue := range []string{"dark", "kettle-light"} {
		panel := NewSettingsPanel()
		panel.LoadConfig(&session.UserConfig{Theme: themeValue})
		// Short enough to force scroll-windowing for every cursor position.
		panel.SetSize(100, 20)
		panel.Show()

		prev := -1
		for cursor := 0; cursor < settingsCount; cursor++ {
			panel.cursor = cursor
			view := panel.View()
			if strings.TrimSpace(view) == "" {
				t.Fatalf("theme %s: empty view at cursor %d", themeValue, cursor)
			}
			line := panel.scrollOffset
			if line < prev {
				t.Errorf("theme %s: scroll offset went backwards at cursor %d (%d < %d)",
					themeValue, cursor, line, prev)
			}
			prev = line
		}
	}
}
