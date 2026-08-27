package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// applyHousePalette wires a house palette (internal/theme) into the active
// color globals. Callers must hold themeMu; InitTheme does.
func applyHousePalette(p theme.Palette) {
	currentTheme = Theme(p.Name)
	ColorBg = lipgloss.Color(p.Bg)
	ColorSurface = lipgloss.Color(p.Surface)
	ColorBorder = lipgloss.Color(p.Border)
	ColorText = lipgloss.Color(p.Text)
	ColorTextDim = lipgloss.Color(p.TextDim)
	ColorAccent = lipgloss.Color(p.Accent)
	ColorPurple = lipgloss.Color(p.Purple)
	ColorCyan = lipgloss.Color(p.Cyan)
	ColorGreen = lipgloss.Color(p.Green)
	ColorYellow = lipgloss.Color(p.Yellow)
	ColorOrange = lipgloss.Color(p.Orange)
	ColorRed = lipgloss.Color(p.Red)
	ColorComment = lipgloss.Color(p.Comment)
}

// ThemeIsLight reports whether the active theme has a light background.
// Use this instead of comparing against ThemeLight: a house palette can be
// light without being the built-in "light" theme.
func ThemeIsLight() bool {
	if currentTheme == ThemeLight {
		return true
	}
	if p, ok := theme.Get(string(currentTheme)); ok {
		return p.Light
	}
	return false
}

// themeNoteFor returns the caption to show under the theme picker for the
// theme at the given radio index (empty when there is nothing to say).
func themeNoteFor(index int) string {
	if index < 0 || index >= len(themeValues) {
		return ""
	}
	if p, ok := theme.Get(themeValues[index]); ok {
		return p.Note
	}
	return ""
}
