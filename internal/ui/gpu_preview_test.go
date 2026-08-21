package ui

import "testing"

func TestGpuTagForTool(t *testing.T) {
	cases := map[string]string{
		"local-ornith":     "ornith:9b",
		"ornith":           "ornith:9b",
		"ornith-35":        "ornith:35b",
		"local-ornith-35b": "ornith:35b",
		"gemma4":           "gemma4:26b",
		"local-gemma4":     "gemma4:26b",
		"claude":           "",
		"grok":             "",
		"":                 "",
	}
	for tool, want := range cases {
		if got := gpuTagForTool(tool); got != want {
			t.Errorf("gpuTagForTool(%q)=%q want %q", tool, got, want)
		}
	}
}
