package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestJoinPillsFlow_WrapsBetweenPillsNotMidEntry(t *testing.T) {
	// Build a few wide pills so a narrow budget must wrap.
	style := lipgloss.NewStyle().Padding(0, 1)
	pills := []string{
		style.Render("claude"),
		style.Render("grok"),
		style.Render("agy"),
		style.Render("gemini"),
		style.Render("codex"),
		style.Render("ornith"),
	}
	// Budget fits ~2 typical pills (~8–12 cells each with padding).
	out := joinPillsFlow(pills, 24)
	if out == "" {
		t.Fatal("empty output")
	}
	// Every full pill label must appear intact (no mid-word split of the names).
	for _, name := range []string{"claude", "grok", "agy", "gemini", "codex", "ornith"} {
		if !strings.Contains(out, name) {
			t.Fatalf("missing intact pill %q in:\n%s", name, out)
		}
	}
	// Must use more than one row at this width.
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected wrap onto multiple rows, got single line: %q", out)
	}
	// Each visual row (after strip indent) should be <= budget.
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimPrefix(line, "  ")
		if w := lipgloss.Width(line); w > 24 {
			t.Fatalf("row wider than budget (%d > 24): %q", w, line)
		}
	}
}

func TestJoinPillsFlow_Empty(t *testing.T) {
	if got := joinPillsFlow(nil, 40); got != "" {
		t.Fatalf("got %q", got)
	}
}
