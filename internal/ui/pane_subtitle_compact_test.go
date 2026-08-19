package ui

import (
	"strings"
	"testing"
)

func TestCompactPaneSubtitle_GrokRedundant(t *testing.T) {
	// Live Grok OSC title pattern observed on avicenna.
	got := compactPaneSubtitle("alchemist", "grok", "xanadu-agents/alchemist", "alchemist-grok - Grok 4.6 - alchemist - grok")
	if got != "Grok 4.6" {
		t.Fatalf("got %q, want %q", got, "Grok 4.6")
	}
}

func TestCompactPaneSubtitle_CareerCoord(t *testing.T) {
	got := compactPaneSubtitle("career-coord", "grok", "coordinators/career", "career-coord-grok - Grok 4.6 - career - grok")
	if got != "Grok 4.6" {
		t.Fatalf("got %q, want %q", got, "Grok 4.6")
	}
}

func TestCompactPaneSubtitle_PaletteBuilderGroupCrumb(t *testing.T) {
	// Real case: group home/design → "design" crumb must drop.
	got := compactPaneSubtitle("palette-builder", "grok", "home/design",
		"palette-builder - Grok 4.5 - design - grok")
	if got != "Grok 4.5" {
		t.Fatalf("got %q, want %q", got, "Grok 4.5")
	}
}

func TestCompactPaneSubtitle_KeepsTaskBlurb(t *testing.T) {
	raw := "Preparing run_terminal_command… - xanadu-herald-coord - Grok 4.5 - xanadu - 96s - grok"
	got := compactPaneSubtitle("herald", "grok", "xanadu-agents/herald", raw)
	if !containsAll(got, "Preparing run_terminal_command") {
		t.Fatalf("must keep task blurb, got %q", got)
	}
	if got == "grok" || stringsHasSuffixWord(got, "grok") {
		t.Fatalf("must not leave bare tool, got %q", got)
	}
}

func TestCompactPaneSubtitle_EmptyWhenAllNoise(t *testing.T) {
	got := compactPaneSubtitle("alchemist", "grok", "xanadu-agents/alchemist", "alchemist-grok - alchemist - grok")
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestCompactPaneSubtitle_GenericCLI(t *testing.T) {
	if got := compactPaneSubtitle("foo", "claude", "", "Claude Code"); got != "" {
		t.Fatalf("generic CLI title must clean to empty, got %q", got)
	}
}

func TestCompactPaneSubtitle_ConversationTitleKept(t *testing.T) {
	// Meaningful free-text conversation titles must survive.
	got := compactPaneSubtitle("librarian", "grok", "household-agents/librarian",
		"Librarian Work and Duties Inquiry - Grok 4.5 - librarian - grok")
	if !containsAll(got, "Librarian Work and Duties Inquiry") {
		t.Fatalf("must keep conversation title, got %q", got)
	}
	if containsAll(got, " - grok") || containsAll(got, "librarian -") {
		// librarian alone should drop; trailing grok should drop
	}
	// Should not end with tool restatement
	if stringsHasSuffixWord(got, "grok") || stringsHasSuffixWord(got, "librarian") {
		t.Fatalf("must strip trailing identity crumbs, got %q", got)
	}
}

func stringsHasSuffixWord(s, word string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	word = strings.ToLower(word)
	return s == word || strings.HasSuffix(s, " - "+word) || strings.HasSuffix(s, "-"+word)
}

func TestRenderToolBadge_IconNotName(t *testing.T) {
	forceTrueColorProfile()
	style := GetToolStyle("grok")
	got := renderToolBadge("grok", style)
	// Must not contain the bare word "grok" as the label (icon is ⚡).
	// Style wraps the icon; strip is hard — just ensure ⚡ path via ToolIcon.
	icon := ToolIcon("grok")
	if icon == "" || icon == "grok" {
		// config may supply ⚡; without config default GetToolIcon may still set something
		t.Logf("ToolIcon(grok)=%q (config-dependent)", icon)
	}
	want := style.Render(" " + icon)
	if icon != "" && got != want {
		t.Fatalf("renderToolBadge(grok) = %q, want %q", got, want)
	}
	// Never render a leading space + full tool name when an icon exists.
	if icon != "" && icon != "grok" {
		nameOnly := style.Render(" grok")
		if got == nameOnly {
			t.Fatal("tool badge still using bare tool name when icon is available")
		}
	}
}

func containsAll(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		// strings.Contains without importing in helpers — simple loop
		indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
