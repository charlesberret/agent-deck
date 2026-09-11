package session

import (
	"strings"
	"testing"
)

// Custom tools (oll-mid, local-ornith, …) spawn through buildGenericCommand.
// The wrapper reads AGENTDECK_INSTANCE_ID at exec to pin Pi's --session-dir.
// tmux SetEnvironment after Start is too late for that first process.
func TestBuildGenericCommand_InjectsAgentdeckEnv(t *testing.T) {
	_ = isolateToolConfigHome(t)

	inst := &Instance{
		ID:      "3a05056a-1786288773",
		Title:   "alchemist",
		Tool:    "oll-mid",
		Command: "/Users/charles/.local/bin/oll-mid",
	}
	got := inst.buildGenericCommand(inst.Command)
	if !strings.Contains(got, "AGENTDECK_INSTANCE_ID=3a05056a-1786288773") {
		t.Fatalf("generic spawn must export AGENTDECK_INSTANCE_ID before the wrapper, got %q", got)
	}
	if !strings.Contains(got, "AGENTDECK_TOOL=oll-mid") {
		t.Fatalf("generic spawn must export AGENTDECK_TOOL, got %q", got)
	}
	if !strings.Contains(got, "AGENTDECK_PROFILE=") {
		t.Fatalf("generic spawn must export AGENTDECK_PROFILE, got %q", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), inst.Command) {
		t.Fatalf("wrapper command should remain the tail, got %q", got)
	}
}
