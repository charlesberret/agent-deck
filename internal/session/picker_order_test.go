package session

import "testing"

func TestBuildPickerPresets_HouseOrder(t *testing.T) {
	t.Log("start")
	custom := map[string]ToolDef{
		"grok":     {Command: "grok"},
		"agy":      {Command: "agy"},
		"ornith":   {Command: "ornith"},
		"penlight": {Command: "penlight"},
		"oll-mid":  {Command: "oll-mid"},
		"wk-glm":   {Command: "wk-glm"},
	}
	t.Log("init")
	r := InitFiltered(custom, false, []string{"cursor", "antigravity"})
	t.Log("build")
	got := r.BuildPickerPresets()
	t.Logf("got=%v", got)
	if len(got) < 5 {
		t.Fatalf("too short: %v", got)
	}
	if got[0] != "claude" {
		t.Fatalf("first=%q want claude full=%v", got[0], got)
	}
	if got[len(got)-1] != "" {
		t.Fatalf("last not shell: %v", got)
	}
	// grok after claude
	idx := map[string]int{}
	for i, n := range got {
		idx[n] = i
	}
	if idx["grok"] != 1 {
		t.Fatalf("grok at %d want 1: %v", idx["grok"], got)
	}
	if idx["opencode"] < idx["wk-glm"] {
		t.Fatalf("opencode before wk: %v", got)
	}
}
