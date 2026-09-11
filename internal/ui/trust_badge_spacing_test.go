package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Every trust-cluster token must be separated by exactly one ASCII space.
// Plain forms:
//
//	" ↑ ▣"       egress + brand
//	" ↑ ▣ ×"     egress + brand + flag
//	" ▣"         brand only (no egress registered — rare)
//	" ▣ ×"       brand + flag
func TestToolBadge_ExactlyOneSpaceBetweenTokens(t *testing.T) {
	forceTrueColorProfile()
	resetTrustEgressForTest()
	resetTrustSubjectForTest()
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", filepathMissingRegistry(t))

	style := GetToolStyle("claude")
	plain := ansi.Strip(renderToolBadge("claude", style))
	assertSingleSpacedTokens(t, plain, "unflagged claude")

	// Force a flag via degraded registry → "?"
	flagged := ansi.Strip(renderToolBadgeForPath("claude", []string{"/anywhere"}, style))
	assertSingleSpacedTokens(t, flagged, "flagged claude")
	if !strings.HasSuffix(flagged, " ?") && !strings.Contains(flagged, " ?") {
		// flag is last token
		parts := strings.Fields(strings.TrimSpace(flagged))
		if len(parts) < 2 || parts[len(parts)-1] != "?" {
			t.Fatalf("expected trailing ? flag, got %q", flagged)
		}
	}
}

func TestToolBadge_NoFlushEgressBrand(t *testing.T) {
	forceTrueColorProfile()
	resetTrustEgressForTest()
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", filepathMissingRegistry(t))

	plain := ansi.Strip(renderToolBadge("claude", GetToolStyle("claude")))
	if strings.Contains(plain, "↑▣") || strings.Contains(plain, "↑›") {
		t.Fatalf("egress must not sit flush against brand icon, got %q", plain)
	}
	if !strings.Contains(plain, "↑ ") {
		t.Fatalf("expected '↑ ' before brand, got %q", plain)
	}
}

func filepathMissingRegistry(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/missing.yaml"
}

func assertSingleSpacedTokens(t *testing.T, plain, label string) {
	t.Helper()
	if plain == "" {
		t.Fatalf("%s: empty badge", label)
	}
	if !strings.HasPrefix(plain, " ") {
		t.Fatalf("%s: missing leading title|badge space: %q", label, plain)
	}
	body := plain[1:] // after leading space
	if body == "" {
		t.Fatalf("%s: empty after leading space", label)
	}
	// No leading/trailing double spaces; no flush (would mean zero spaces in join).
	if strings.Contains(plain, "  ") {
		t.Fatalf("%s: double space in badge: %q", label, plain)
	}
	// Re-join Fields with single spaces must equal TrimSpace(plain) — proves
	// tokens were separated by exactly one space and nothing else.
	fields := strings.Fields(plain)
	rejoined := strings.Join(fields, " ")
	if rejoined != strings.TrimSpace(plain) {
		t.Fatalf("%s: irregular spacing: plain=%q fields=%v", label, plain, fields)
	}
	if len(fields) < 1 {
		t.Fatalf("%s: no tokens", label)
	}
}
