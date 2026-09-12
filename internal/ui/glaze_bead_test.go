package ui

// The chrome-only glaze, and the one field the deck spends it on.
//
// Kettle's glaze packs carry a "body-voiced" enamel (Cerise cherry, Nectar
// amber, Artichaut green) whose hue collides with a status signal, so it may
// never colour text. The design law: "the enamel fills a field only where the
// field is transient or small (cursor, bead, dot); a persistent field is a
// shade." The deck's only such field is the leading selection bead on the
// cursor row, drawn by SessionSelectionPrefix.
//
// Seam A (see TUI_TESTS.md): the observable is one render function's output
// for one row, so no teatest runtime is needed.

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// withTrueColorProfile forces (and then restores) lipgloss' global colour
// profile. Go tests do not run under a TTY, so auto-detect lands on Ascii and
// strips every escape, which would make the assertions below vacuous.
//
// Restore matters here and does not in issue391_tui_test.go: that file sorts
// after home_test.go, whose footer assertions match on un-escaped substrings
// and break if a truecolor profile is still latched on when they run.
func withTrueColorProfile(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// TestHousePalettesPopulateColorGlaze pins the fallback wiring: every house
// palette must land a usable glaze in the global, whether it declares one (all
// of them do today) or the generator omits it for a pack with no enamel.
func TestHousePalettesPopulateColorGlaze(t *testing.T) {
	defer InitTheme("dark")

	for _, p := range theme.All() {
		InitTheme(p.Name)
		if want := lipgloss.Color(p.GlazeColor()); ColorGlaze != want {
			t.Errorf("%s: ColorGlaze = %q, want %q", p.Name, ColorGlaze, want)
		}
		if p.Glaze == "" && ColorGlaze != ColorAccent {
			t.Errorf("%s: no Glaze declared, so ColorGlaze should equal ColorAccent (%q), got %q",
				p.Name, ColorAccent, ColorGlaze)
		}
	}
}

// TestBuiltInThemesFallBackToAccentGlaze keeps the built-in Tokyo Night
// schemes on their accent: they have no enamel of their own, and a zero-valued
// ColorGlaze would render the bead in the terminal's default foreground.
func TestBuiltInThemesFallBackToAccentGlaze(t *testing.T) {
	defer InitTheme("dark")

	for _, name := range []string{"dark", "light"} {
		InitTheme(name)
		if ColorGlaze != ColorAccent {
			t.Errorf("%s: ColorGlaze = %q, want ColorAccent %q", name, ColorGlaze, ColorAccent)
		}
	}
}

// TestSelectedRowBeadCarriesTheGlaze is the load-bearing pin. It installs a
// glaze of its own rather than trusting a shipped palette, for two reasons:
// the assertion then runs against a known colour instead of whatever the
// generator last emitted, and a pack whose glaze happens to equal its accent
// would make the test vacuous — it could not tell ColorGlaze from ColorAccent.
// Under a distinct glaze it proves:
//
//  1. the selected row's leading bead is drawn in the glaze, and
//  2. nothing else on the row is — the title keeps its accent-backed shade.
//
// If someone re-points SessionSelectionPrefix at ColorAccent, (1) fails. If
// someone spends the glaze on text or a row background, (2) fails.
func TestSelectedRowBeadCarriesTheGlaze(t *testing.T) {
	withTrueColorProfile(t)

	InitTheme("kettle")
	defer InitTheme("dark")

	// Artichaut-like green: a body-voiced enamel, deliberately unlike any
	// colour kettle already uses, so the escape below can only have come from
	// ColorGlaze and not from the accent or a status style. The
	// expected escape is taken from lipgloss itself rather than hand-written,
	// because the colour profile rounds channels on its way to ANSI.
	themeMu.Lock()
	ColorGlaze = lipgloss.Color("#5aa050")
	initStyles()
	themeMu.Unlock()
	// initStyles is re-run by the deferred InitTheme, so the override does not
	// leak into other tests in the package.
	glazeEscape := foregroundEscape(t, ColorGlaze)
	glazeAsBackground := strings.Replace(glazeEscape, "38;2;", "48;2;", 1)

	inst := &session.Instance{ID: "glaze-1", Title: "glaze bead", Tool: "shell", Status: session.StatusIdle}
	snapshot := map[string]sessionRenderState{
		inst.ID: {status: session.StatusIdle, tool: "shell"},
	}
	item := session.Item{
		Type:          session.ItemTypeSession,
		Session:       inst,
		Level:         1,
		Path:          "test",
		IsLastInGroup: true,
	}

	render := func(selected bool) string {
		h := &Home{width: 140}
		var b strings.Builder
		h.renderSessionItem(&b, item, selected, snapshot, h.width)
		return b.String()
	}

	selected, unselected := render(true), render(false)

	if !strings.Contains(selected, glazeEscape) {
		t.Errorf("selected row does not carry the glaze (%s); the bead should be the one field that does:\n%q",
			glazeEscape, selected)
	}
	if strings.Contains(unselected, glazeEscape) {
		t.Errorf("unselected row carries the glaze (%s); the enamel belongs to the cursor row only:\n%q",
			glazeEscape, unselected)
	}
	// The bead is a leading one-cell marker, not a wash over the row: the
	// glaze must appear exactly once, and ahead of every other styled cell.
	if n := strings.Count(selected, glazeEscape); n != 1 {
		t.Errorf("glaze escape appears %d times in the selected row, want 1 (bead only):\n%q", n, selected)
	}
	if i, j := strings.Index(selected, glazeEscape), strings.Index(selected, "48;2;"); j >= 0 && i > j {
		t.Errorf("glaze escape at %d comes after the row's first painted cell at %d; the bead leads the row:\n%q",
			i, j, selected)
	}
	// The enamel is foreground-only: a background fill is a persistent field,
	// which the law says must be a shade.
	if strings.Contains(selected, glazeAsBackground) {
		t.Errorf("glaze used as a background; a persistent field is a shade, not enamel:\n%q", selected)
	}
}

// foregroundEscape returns the ANSI sequence lipgloss emits for c as a
// foreground under the active colour profile, so assertions do not have to
// guess how the profile rounds channels.
func foregroundEscape(t *testing.T, c lipgloss.Color) string {
	t.Helper()
	rendered := lipgloss.NewStyle().Foreground(c).Render("x")
	start := strings.Index(rendered, "38;2;")
	if start < 0 {
		t.Fatalf("colour profile emitted no truecolor foreground for %q: %q", c, rendered)
	}
	seg := rendered[start:]
	end := strings.Index(seg, "m")
	if end < 0 {
		t.Fatalf("malformed escape for %q: %q", c, rendered)
	}
	return seg[:end]
}
