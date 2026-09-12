package ui

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

func TestInitThemeAppliesHousePalettes(t *testing.T) {
	defer InitTheme("dark")

	for _, p := range theme.All() {
		InitTheme(p.Name)
		if got := string(GetCurrentTheme()); got != p.Name {
			t.Errorf("after InitTheme(%q), GetCurrentTheme() = %q", p.Name, got)
		}
		if ColorBg != lipgloss.Color(p.Bg) {
			t.Errorf("%s: ColorBg = %q, want %q", p.Name, ColorBg, p.Bg)
		}
		if ColorAccent != lipgloss.Color(p.Accent) {
			t.Errorf("%s: ColorAccent = %q, want %q", p.Name, ColorAccent, p.Accent)
		}
		if ThemeIsLight() != p.Light {
			t.Errorf("%s: ThemeIsLight() = %v, want %v", p.Name, ThemeIsLight(), p.Light)
		}
		// Styles must be rebuilt, not left on the previous palette.
		if BaseStyle.GetBackground() != lipgloss.Color(p.Bg) {
			t.Errorf("%s: BaseStyle background not refreshed", p.Name)
		}
	}
}

func TestInitThemeUnknownFallsBackToDark(t *testing.T) {
	defer InitTheme("dark")
	InitTheme("kettle")
	InitTheme("no-such-theme")
	if GetCurrentTheme() != ThemeDark {
		t.Errorf("unknown theme should fall back to dark, got %q", GetCurrentTheme())
	}
	if ColorBg != darkColors.Bg {
		t.Errorf("ColorBg = %q, want dark %q", ColorBg, darkColors.Bg)
	}
}

func TestThemeRadioListsStayAligned(t *testing.T) {
	if len(themeNames) != len(themeValues) {
		t.Fatalf("themeNames (%d) and themeValues (%d) out of sync", len(themeNames), len(themeValues))
	}
	for i, want := range []string{"dark", "light", "system"} {
		if themeValues[i] != want {
			t.Errorf("themeValues[%d] = %q, want %q — built-in indices are load-bearing", i, themeValues[i], want)
		}
	}
	for _, name := range theme.Names() {
		if themeIndex(name) < 3 {
			t.Errorf("house theme %q resolved to a built-in index", name)
		}
	}
	if themeIndex("garbage") != 0 {
		t.Error("unknown theme should map to index 0 (dark)")
	}
}

func TestThemeNoteSurfacesWIPPalettes(t *testing.T) {
	for _, p := range theme.All() {
		got := themeNoteFor(themeIndex(p.Name))
		if got != p.Note {
			t.Errorf("themeNoteFor(%s) = %q, want %q", p.Name, got, p.Note)
		}
	}
	for i := range []string{"dark", "light", "system"} {
		if note := themeNoteFor(i); note != "" {
			t.Errorf("built-in theme at index %d carries a note: %q", i, note)
		}
	}
	if themeNoteFor(-1) != "" || themeNoteFor(len(themeValues)+5) != "" {
		t.Error("out-of-range index should yield no note")
	}
}

func TestThemeRowFitsSettingsDialog(t *testing.T) {
	// The real constraint is not the pill budget — it is the settings dialog's
	// content width. SettingsPanel.View renders at dialogWidth 64 with
	// Padding(1, 2) (border sits outside Width), so content is dialogWidth-4
	// cells. Every rendered row is written after a two-space indent, first row
	// from the caller and continuation rows from joinPillsFlow, so the thing
	// that must fit is 2 + width(line).
	const (
		dialogWidth      = 64
		settingsHorizPad = 2 // dialogStyle: Padding(1, 2)
		indent           = 2 // content.WriteString("  " + themeRow)
	)
	contentWidth := dialogWidth - 2*settingsHorizPad

	row := radioGroupFlow(themeNames, 0, pillsInnerWidth(dialogWidth))
	for i, line := range strings.Split(row, "\n") {
		if w := indent + lipgloss.Width(line); w > contentWidth {
			t.Errorf("theme row line %d occupies %d cols (indent %d + %d), settings content width is %d: %q",
				i, w, indent, lipgloss.Width(line), contentWidth, line)
		}
	}

	// And the budget itself must leave room for the indent it promises to pay
	// for: an off-by-two in pillsInnerWidth shows up here even though the
	// settings panel's lighter padding would otherwise absorb it.
	const pillsHorizPad = 4 // pillsInnerWidth documents Padding(2, 4)
	if got, want := pillsInnerWidth(dialogWidth), dialogWidth-2*pillsHorizPad-indent; got != want {
		t.Errorf("pillsInnerWidth(%d) = %d, want %d (content %d less the %d-cell indent every row carries)",
			dialogWidth, got, want, dialogWidth-2*pillsHorizPad, indent)
	}
}
