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
	// The settings dialog renders at width 64; anything wider than the inner
	// budget has to wrap between options rather than spill.
	const dialogWidth = 64
	budget := pillsInnerWidth(dialogWidth)
	row := radioGroupFlow(themeNames, 0, budget)
	for i, line := range strings.Split(row, "\n") {
		if w := lipgloss.Width(line); w > budget {
			t.Errorf("theme row line %d is %d cols, budget %d: %q", i, w, budget, line)
		}
	}
}
