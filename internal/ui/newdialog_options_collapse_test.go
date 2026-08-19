package ui

import (
	"strings"
	"testing"
)

// Collapsed tool-options: full Claude panel stays hidden until focusOptions.
func TestNewDialog_ToolOptionsCollapsedUntilFocused(t *testing.T) {
	d := NewNewDialog()
	d.SetSize(100, 50)
	d.Show()
	d.SetDefaultTool("claude")
	// Focus on name (not options)
	d.focusIndex = d.indexOf(focusName)
	if d.focusIndex < 0 {
		t.Fatal("focusName missing")
	}
	d.updateFocus()

	view := d.View()
	if !strings.Contains(view, "Claude Options") {
		t.Fatalf("collapsed row should still name the section:\n%s", view)
	}
	if !strings.Contains(view, "expand") {
		t.Fatalf("collapsed row should hint expand:\n%s", view)
	}
	// Expanded panel content (checkboxes) should NOT appear while collapsed
	if strings.Contains(view, "Skip permissions") {
		t.Fatalf("full Claude options should not render while unfocused:\n%s", view)
	}

	// Arrow onto options section
	idx := d.indexOf(focusOptions)
	if idx < 0 {
		t.Fatal("focusOptions should be in targets when tool has options")
	}
	d.focusIndex = idx
	d.updateFocus()
	view = d.View()
	if !strings.Contains(view, "Skip permissions") {
		t.Fatalf("full Claude options should expand when focused:\n%s", view)
	}
}

func TestNewDialog_ToolOptionsCollapsedWhenSwitchingTools(t *testing.T) {
	d := NewNewDialog()
	d.SetSize(100, 50)
	d.Show()
	// Stay on command picker while flipping tools
	d.focusIndex = d.indexOf(focusCommand)
	d.updateFocus()

	d.SetDefaultTool("claude")
	vClaude := d.View()
	d.SetDefaultTool("codex")
	vCodex := d.View()
	// shell has no tool-options panel (always in presets)
	d.SetDefaultTool("")
	vShell := d.View()

	// claude/codex show collapsed one-liner only while focus stays on command
	if strings.Contains(vClaude, "Skip permissions") {
		t.Fatal("claude options expanded while still on command field")
	}
	if !strings.Contains(vClaude, "Claude Options") {
		t.Fatal("claude collapsed title missing")
	}
	if !strings.Contains(vCodex, "Codex Options") {
		t.Fatalf("codex collapsed title missing:\n%s", vCodex)
	}
	if strings.Contains(vCodex, "bypass approvals") {
		t.Fatal("codex YOLO panel should stay collapsed on command field")
	}
	// shell has no tool-options section
	if strings.Contains(vShell, "Claude Options") || strings.Contains(vShell, "Codex Options") {
		t.Fatalf("shell should not show tool options section:\n%s", vShell)
	}
}
