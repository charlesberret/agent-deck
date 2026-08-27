// Package theme holds the house color palettes that agent-deck offers in
// addition to the two built-in Tokyo Night schemes ("dark" and "light").
//
// It is a leaf package on purpose: internal/ui, internal/session and
// internal/tmux all need to agree on which theme names are valid and what
// chrome a theme wants, and none of them can import each other. Colors are
// stored as "#rrggbb" strings so tmux can use them verbatim and the UI can
// wrap them in lipgloss.Color.
//
// Palette provenance lives in ~/cloud/sync/design/palettes — a value that is
// not in the source palette is marked "derived" with the derivation.
package theme

// Palette is one named color scheme. The thirteen color roles mirror the
// built-in dark/light structs in internal/ui/styles.go; the Tmux* fields
// describe the chrome agent-deck paints onto its tmux sessions.
type Palette struct {
	Name    string // config value, e.g. "wildcherry"
	Display string // settings-panel label, e.g. "Wild Cherry"
	Light   bool   // true for light-background palettes

	// WIP marks a palette whose source is still being revised upstream of
	// agent-deck. Note is shown under the theme picker when it is selected,
	// so the deck says out loud that these colors may move.
	WIP  bool
	Note string

	Bg      string
	Surface string
	Border  string
	Text    string
	TextDim string
	Accent  string
	Purple  string
	Cyan    string
	Green   string
	Yellow  string
	Orange  string
	Red     string
	Comment string

	// TmuxWindowStyle is the tmux window-style value. "default" lets the
	// terminal's own background (and any transparency or blur it applies)
	// show through the pane; an explicit "bg=#rrggbb" paints over it.
	TmuxWindowStyle string
	TmuxStatusBg    string
	TmuxStatusFg    string
	TmuxHint        string
}

// palettes is the house registry. Built-in "dark"/"light" are NOT here —
// they stay in internal/ui/styles.go so upstream owns them.
var palettes = map[string]Palette{
	// Wild Cherry — the Ghostty colorscheme avicenna runs, so the deck and
	// the terminal underneath it read as one surface. Values taken from
	// ~/.config/ghostty/config (Charles's in-use variant, which differs
	// slightly from Ghostty's shipped "Wild Cherry" theme file).
	"wildcherry": {
		Name:    "wildcherry",
		Display: "Wild Cherry",

		Bg:      "#1f1626", // ghostty background
		Surface: "#2c2433", // derived: bg mixed 6% toward white
		Border:  "#3c2059", // derived: bg mixed 28% toward palette 4 (#883bdc)
		Text:    "#d9faff", // ghostty foreground
		TextDim: "#85939d", // derived: fg mixed 45% toward bg
		Accent:  "#d93f85", // palette 1 — the cherry; selection + titles
		Purple:  "#883bdc", // palette 4
		Cyan:    "#009cc9", // palette 8
		Green:   "#2ab250", // palette 2
		Yellow:  "#ffd16e", // palette 3
		Orange:  "#eac066", // palette 11
		Red:     "#ff919d", // palette 14 — brightest red-family, reads as alarm
		Comment: "#85939d", // = TextDim

		TmuxWindowStyle: "default", // keep Ghostty's opacity/blur intact
		TmuxStatusBg:    "#2c2433",
		TmuxStatusFg:    "#d9faff",
		TmuxHint:        "#85939d",
	},

	// Kettle — the product palette, dark mode. Source:
	// ~/cloud/sync/design/palettes/library/kettle.json (modes.dark.roles).
	// "Quiet instrument desk", gold accent, coral warn.
	"kettle": {
		Name:    "kettle",
		Display: "Kettle",

		Bg: "#16161e", // role ground
		// Surface is derived (ground mixed 25% toward line): kettle.json keeps
		// panel == ground and flags that flatness as an improvement target,
		// while agent-deck paints Surface as a real background.
		Surface: "#1d1d26",
		Border:  "#32323f", // role line
		Text:    "#d8d8e0", // role ink
		TextDim: "#87878f", // role muted
		Accent:  "#c8a24a", // role accent (gold)
		Purple:  "#9a76c2", // role magenta
		Cyan:    "#389bb4", // role cyan
		Green:   "#7dba6f", // role good
		Yellow:  "#d4b86a", // role yellow_bright
		Orange:  "#e8928a", // role red_bright
		Red:     "#d4756b", // role warn (coral)
		Comment: "#87878f", // = TextDim

		TmuxWindowStyle: "default",
		TmuxStatusBg:    "#1d1d26",
		TmuxStatusFg:    "#d8d8e0",
		TmuxHint:        "#87878f",
	},

	// Kettle Light — same product palette, light mode. Source:
	// design/palettes/library/kettle.json (modes.light.roles), rebuilt
	// 2026-08-13 in OKLCH against WCAG floors (contrast_matrix in that file).
	//
	// WIP: kettle.json still lists open improvements for this mode (iface.panel2
	// indexes ansi-0 in the terminal map; light bright-* steps are new; the whole
	// light pass is unpromoted to kettle-core pending dual-mode visual QA). Re-read
	// modes.light.roles and update these values when that pass lands.
	"kettle-light": {
		Name:    "kettle-light",
		Display: "Kettle Light",
		Light:   true,
		WIP:     true,
		Note:    "Kettle Light is a work in progress — colors may be revised.",

		Bg:      "#dfe1e8", // role ground
		Surface: "#ebecf2", // role panel (soft elevated chrome)
		Border:  "#bcbdc6", // role line — dedicated hairline, darker than panel
		Text:    "#181a23", // role ink
		TextDim: "#5b5d66", // role muted (5.02:1 on ground)
		Accent:  "#1b4979", // role accent / link (blue; light mode moves off gold)
		Purple:  "#633f81", // role magenta
		Cyan:    "#006276", // role cyan
		Green:   "#00591a", // role good
		Yellow:  "#855100", // role gold
		Orange:  "#b33830", // role red_bright
		Red:     "#8e1904", // role warn
		Comment: "#5b5d66", // = TextDim; role faint (#787a82) is too pale for hints

		// Light themes must paint the pane: a light TUI inside a dark terminal
		// otherwise shows agent output on the terminal's dark background.
		TmuxWindowStyle: "bg=#dfe1e8",
		TmuxStatusBg:    "#ebecf2",
		TmuxStatusFg:    "#181a23",
		TmuxHint:        "#5b5d66",
	},
}

// Get returns the house palette registered under name.
func Get(name string) (Palette, bool) {
	p, ok := palettes[name]
	return p, ok
}

// IsHouse reports whether name is a house theme (as opposed to the built-in
// "dark"/"light"/"system" values, or garbage).
func IsHouse(name string) bool {
	_, ok := palettes[name]
	return ok
}

// order is the display order of the house themes in the picker: the terminal
// scheme avicenna actually runs first, then the product palette and its light
// variant. Every key in palettes must appear here (asserted by the tests).
var order = []string{"wildcherry", "kettle", "kettle-light"}

// Names returns the house theme config values in display order.
func Names() []string {
	out := make([]string, 0, len(order))
	for _, name := range order {
		if _, ok := palettes[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

// All returns the house palettes in Names() order.
func All() []Palette {
	names := Names()
	out := make([]Palette, 0, len(names))
	for _, name := range names {
		out = append(out, palettes[name])
	}
	return out
}
