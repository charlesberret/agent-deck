package session

// pickerHouseOrder is the avicenna-preferred new-session tool order
// (local/avicenna branch). Empty string is shell and always last.
//
// Groups:
//  1. daily drivers — claude, grok, agy, gemini
//  2. local — ornith, penlight
//  3. oll-* grades
//  4. wk-* personas
//  5. codex, copilot, then barely-used (crush…aider)
//  6. shell
//
// Duplicates not listed here (local-ornith, local-penlight, antigravity) should
// stay in [ui].hidden_tools so they do not reappear alphabetically at the end.
var pickerHouseOrder = []string{
	"claude", "grok", "agy", "gemini",
	"ornith", "penlight",
	"oll-light", "oll-mid", "oll-heavy", "oll-hard-heavy", "oll-vision", "oll-pro",
	"wk-glm", "wk-granite", "wk-kimi-code", "wk-m3",
	// cloud-adjacent then barely-used: codex immediately before copilot
	"codex", "copilot", "crush", "hermes", "deepseek", "pi", "opencode", "cursor", "aider",
	// shell appended as "" below
}

// BuildPickerPresets returns the ordered tool-picker list for the process
// registry (see Registry.BuildPickerPresets).
func BuildPickerPresets() []string {
	return currentRegistry().BuildPickerPresets()
}

// BuildPickerPresets returns the ordered tool-picker list: house preference
// first, then any remaining custom tools (sorted), then remaining builtins,
// then shell (""). Applies hidden_tools + show_only_installed_tools.
func (r *Registry) BuildPickerPresets() []string {
	if r == nil {
		return []string{""}
	}

	available := map[string]bool{"": true}
	for _, d := range r.All() {
		// All() uses tool name as Command for builtins
		available[d.Command] = true
	}
	// shell is listed as "shell" in All() but the picker stores ""
	available["shell"] = true
	for _, n := range r.CustomNames() {
		available[n] = true
	}

	seen := make(map[string]bool)
	out := make([]string, 0, len(available))

	for _, name := range pickerHouseOrder {
		if name == "" || name == "shell" {
			continue
		}
		if !available[name] || seen[name] {
			continue
		}
		out = append(out, name)
		seen[name] = true
	}

	// Remaining custom tools (CustomNames is already sorted)
	for _, n := range r.CustomNames() {
		if seen[n] {
			continue
		}
		out = append(out, n)
		seen[n] = true
	}

	// Remaining builtins not in house order (except shell)
	for _, d := range r.All() {
		n := d.Command
		if n == "" || n == "shell" || seen[n] {
			continue
		}
		out = append(out, n)
		seen[n] = true
	}

	// Shell last (picker slot is empty command)
	out = append(out, "")

	return r.FilterVisibleNames(out)
}
