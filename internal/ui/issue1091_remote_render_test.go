package ui

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// Force TrueColor so lipgloss emits actual ANSI escapes during tests;
// otherwise the auto-detect falls back to Ascii under `go test` and
// strips every color, making the "tool style applied" assertion vacuous.
// The forceTrueColorProfile helper itself lives in issue391_tui_test.go.

// TestIssue1091_RemoteSession_ToolColorMatchesLocal asserts that a remote
// session with Tool="claude" renders the tool label using the same
// brand-specific style (orange) as a local claude session — NOT the
// generic DimStyle (gray) that the buggy renderer used.
//
// Bug: PR #1073 (v1.9.22) propagated the Tool field over the wire so
// remote sessions know they're "claude", but renderRemoteSessionItem
// still rendered the tool label with DimStyle, dropping the
// claude-specific color (Anthropic orange). Reported by @ddorman-dn in
// #1091 with a screenshot showing colorless "claude"/"shell" labels.
//
// Fix: renderRemoteSessionItem must use GetToolStyle(rs.Tool) for the
// tool label, matching the local renderSessionItem path at home.go:11812.
func TestIssue1091_RemoteSession_ToolColorMatchesLocal(t *testing.T) {
	forceTrueColorProfile()

	home := NewHome()
	home.width = 100
	home.height = 30

	remote := session.RemoteSessionInfo{
		ID:         "remote-claude-1",
		Title:      "my-claude-session",
		Status:     "running",
		Tool:       "claude",
		RemoteName: "myserver",
	}
	item := session.Item{
		Type:          session.ItemTypeRemoteSession,
		RemoteSession: &remote,
		RemoteName:    "myserver",
	}

	var b strings.Builder
	home.renderRemoteSessionItem(&b, item, false)
	rendered := b.String()

	// Brand icon (▣) in tool style — same as local renderToolBadge path.
	expectedToolLabel := renderToolBadge("claude", GetToolStyle("claude"))
	// What the buggy implementation rendered (gray DimStyle + bare name).
	dimToolLabel := DimStyle.Render(" claude")

	if !strings.Contains(rendered, expectedToolLabel) {
		t.Fatalf("remote claude session must render tool badge with GetToolStyle(\"claude\") (orange) "+
			"to match local renderSessionItem.\n"+
			"want substring: %q\n"+
			"got rendered:   %q",
			expectedToolLabel, rendered)
	}

	// Sanity: if the tool style and dim style happen to be identical (some
	// theme edge case), the assertion above already passes — skip the
	// negative check. Otherwise the remote row must NOT use DimStyle for
	// the tool, which is the v1.9.22 bug.
	if expectedToolLabel != dimToolLabel && strings.Contains(rendered, dimToolLabel) {
		t.Fatalf("remote claude session still rendering tool label with DimStyle (gray) — "+
			"this is the #1091 regression. rendered: %q", rendered)
	}
}

// TestIssue1091_RemoteSession_ToolColorAllTools asserts the brand-color
// styling works for every tool agent-deck supports remotely, not just
// claude. Each tool has its own color in ToolStyleCache (claude=orange,
// gemini=purple, codex=cyan, aider=red, etc.).
func TestIssue1091_RemoteSession_ToolColorAllTools(t *testing.T) {
	forceTrueColorProfile()

	tools := []string{"claude", "gemini", "codex", "aider", "opencode", "cursor"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			home := NewHome()
			home.width = 100
			home.height = 30

			remote := session.RemoteSessionInfo{
				ID:         "remote-" + tool,
				Title:      tool + "-session",
				Status:     "running",
				Tool:       tool,
				RemoteName: "myserver",
			}
			item := session.Item{
				Type:          session.ItemTypeRemoteSession,
				RemoteSession: &remote,
				RemoteName:    "myserver",
			}

			var b strings.Builder
			home.renderRemoteSessionItem(&b, item, false)
			rendered := b.String()

			expectedToolLabel := renderToolBadge(tool, GetToolStyle(tool))
			if !strings.Contains(rendered, expectedToolLabel) {
				t.Fatalf("remote %s session must render tool badge with GetToolStyle(%q).\n"+
					"want substring: %q\n"+
					"got rendered:   %q",
					tool, tool, expectedToolLabel, rendered)
			}
		})
	}
}

// TestIssue1091_RemoteSession_SelectedStateUnchanged asserts the fix
// doesn't break selection styling — when the row is selected, the tool
// label should use the selection style (SessionStatusSelStyle), same as
// the local render path.
func TestIssue1091_RemoteSession_SelectedStateUnchanged(t *testing.T) {
	forceTrueColorProfile()

	home := NewHome()
	home.width = 100
	home.height = 30

	remote := session.RemoteSessionInfo{
		ID:         "remote-claude-1",
		Title:      "my-claude-session",
		Status:     "running",
		Tool:       "claude",
		RemoteName: "myserver",
	}
	item := session.Item{
		Type:          session.ItemTypeRemoteSession,
		RemoteSession: &remote,
		RemoteName:    "myserver",
	}

	var b strings.Builder
	home.renderRemoteSessionItem(&b, item, true) // selected=true
	rendered := b.String()

	expectedToolLabel := renderToolBadge("claude", SessionStatusSelStyle)
	if !strings.Contains(rendered, expectedToolLabel) {
		t.Fatalf("selected remote claude session must render tool badge with SessionStatusSelStyle.\n"+
			"want substring: %q\n"+
			"got rendered:   %q",
			expectedToolLabel, rendered)
	}
}
