package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrustEgressGlyph_Builtin(t *testing.T) {
	resetTrustEgressForTest()
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", filepath.Join(t.TempDir(), "missing.yaml"))

	cases := map[string]string{
		"ornith":       "⌂",
		"claude":       "☁",
		"wk-kimi-code": "☁",
		"oll-heavy":    "☁",
		"wk-granite":   "⌂",
		"shell":        "!",
		"GROK":         "☁", // case fold
	}
	for tool, want := range cases {
		if got := TrustEgressGlyph(tool); got != want {
			t.Errorf("TrustEgressGlyph(%q)=%q want %q", tool, got, want)
		}
	}
}

func TestTrustEgressGlyph_RegistryOverrides(t *testing.T) {
	resetTrustEgressForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "reg.yaml")
	body := []byte("tools:\n  claude: { surface: claude-full, egress: vendor, glyph: \"X\" }\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", path)
	if got := TrustEgressGlyph("claude"); got != "X" {
		t.Fatalf("registry override: got %q want X", got)
	}
	// Unlisted tools still fall back to builtin
	if got := TrustEgressGlyph("ornith"); got != "⌂" {
		t.Fatalf("builtin fallback: got %q want ⌂", got)
	}
}

func TestRenderToolBadge_IncludesEgress(t *testing.T) {
	forceTrueColorProfile()
	resetTrustEgressForTest()
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", filepath.Join(t.TempDir(), "missing.yaml"))

	style := GetToolStyle("claude")
	got := renderToolBadge("claude", style)
	if got == "" {
		t.Fatal("empty badge")
	}
	// Must contain cloud egress mark
	if !containsAll(got, "☁") {
		t.Fatalf("expected ☁ egress in badge, got %q", got)
	}
	icon := ToolIcon("claude")
	if icon != "" && !containsAll(got, icon) {
		t.Fatalf("expected brand icon %q in badge, got %q", icon, got)
	}
}

func TestRenderToolBadge_LocalOrnith(t *testing.T) {
	forceTrueColorProfile()
	resetTrustEgressForTest()
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", filepath.Join(t.TempDir(), "missing.yaml"))

	got := renderToolBadge("ornith", GetToolStyle("ornith"))
	if !containsAll(got, "⌂") {
		t.Fatalf("expected ⌂ for ornith, got %q", got)
	}
}
