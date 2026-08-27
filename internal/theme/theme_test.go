package theme

import (
	"regexp"
	"testing"
)

var hexRE = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func TestPalettesAreWellFormed(t *testing.T) {
	for _, p := range All() {
		if p.Name == "" || p.Display == "" {
			t.Errorf("palette %+v missing name or display", p)
		}
		colors := map[string]string{
			"Bg": p.Bg, "Surface": p.Surface, "Border": p.Border,
			"Text": p.Text, "TextDim": p.TextDim, "Accent": p.Accent,
			"Purple": p.Purple, "Cyan": p.Cyan, "Green": p.Green,
			"Yellow": p.Yellow, "Orange": p.Orange, "Red": p.Red,
			"Comment": p.Comment,
			// tmux chrome colors are real hex too (window style is not)
			"TmuxStatusBg": p.TmuxStatusBg, "TmuxStatusFg": p.TmuxStatusFg,
			"TmuxHint": p.TmuxHint,
		}
		for role, hex := range colors {
			if !hexRE.MatchString(hex) {
				t.Errorf("%s.%s = %q, want lowercase #rrggbb", p.Name, role, hex)
			}
		}
		// Surface must be distinguishable from Bg: agent-deck paints Surface
		// as a real background (title bars, selected rows, preview pane).
		if p.Surface == p.Bg {
			t.Errorf("%s: Surface equals Bg (%s) — chrome would be invisible", p.Name, p.Bg)
		}
		if p.TmuxWindowStyle == "" {
			t.Errorf("%s: TmuxWindowStyle empty, want \"default\" or bg=#rrggbb", p.Name)
		}
		// A light TUI inside a dark terminal needs the pane painted, or agent
		// output renders on the terminal's own dark background.
		if p.Light && p.TmuxWindowStyle == "default" {
			t.Errorf("%s: light palette must set an explicit tmux window bg", p.Name)
		}
		// A WIP palette has to say so where the picker can show it.
		if p.WIP && p.Note == "" {
			t.Errorf("%s: WIP palette with no Note — the picker would stay silent", p.Name)
		}
		if !p.WIP && p.Note != "" {
			t.Errorf("%s: Note set on a non-WIP palette (%q)", p.Name, p.Note)
		}
	}
}

func TestRegistryLookups(t *testing.T) {
	if !IsHouse("wildcherry") || !IsHouse("kettle") {
		t.Fatal("expected wildcherry and kettle to be registered")
	}
	for _, builtin := range []string{"dark", "light", "system", ""} {
		if IsHouse(builtin) {
			t.Errorf("IsHouse(%q) = true, want false — built-ins stay in internal/ui", builtin)
		}
	}
	p, ok := Get("wildcherry")
	if !ok || p.Display != "Wild Cherry" {
		t.Errorf("Get(wildcherry) = %+v, %v", p, ok)
	}
	if _, ok := Get("nope"); ok {
		t.Error("Get(nope) should not resolve")
	}
}

func TestNamesAreStable(t *testing.T) {
	a, b := Names(), Names()
	if len(a) != len(palettes) {
		t.Fatalf("Names() returned %d entries, want %d — a palette is missing from order", len(a), len(palettes))
	}
	for _, name := range order {
		if _, ok := palettes[name]; !ok {
			t.Errorf("order lists %q, which is not a registered palette", name)
		}
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Names() unstable: %v vs %v", a, b)
		}
	}
}
