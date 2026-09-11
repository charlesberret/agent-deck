package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const grokResumeID = "019fe0ec-7a6d-7632-bfc8-616d13f6f181"
const grokBinary = "/Users/charles/.local/bin/grok"

// writeGrokToolConfig installs the house grok [tools.grok] shape. Empty
// resumeFlag omits resume_flag and session_id_env (the config that caused
// grok seats to mint a fresh conversation on every start).
func writeGrokToolConfig(t *testing.T, home, resumeFlag string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("\n[tools.grok]\n")
	b.WriteString("command = \"")
	b.WriteString(grokBinary)
	b.WriteString("\"\n")
	if resumeFlag != "" {
		b.WriteString("resume_flag = \"")
		b.WriteString(resumeFlag)
		b.WriteString("\"\n")
		b.WriteString("session_id_env = \"GROK_SESSION_ID\"\n")
	}
	content := b.String()
	for _, dir := range []string{
		filepath.Join(home, ".config", "agent-deck"),
		filepath.Join(home, ".agent-deck"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ClearUserConfigCache()
}

func TestBuildGenericCommand_GrokResumeShape(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "--resume")

	def := GetToolDef("grok")
	if def == nil || def.ResumeFlag != "--resume" || def.SessionIDEnv != "GROK_SESSION_ID" {
		t.Fatalf("GetToolDef(grok) = %+v, want resume_flag=--resume session_id_env=GROK_SESSION_ID", def)
	}

	inst := &Instance{
		ID:               "2ca1962f-1786191695",
		Title:            "agent-deck-admin",
		Tool:             "grok",
		Command:          grokBinary,
		GenericSessionID: grokResumeID,
		tmuxSession:      nil,
	}
	if !inst.CanRestartGeneric() {
		t.Fatal("CanRestartGeneric want true with resume_flag and persisted id")
	}
	cmd := inst.buildGenericCommand(inst.Command)
	want := grokBinary + " --resume " + grokResumeID
	if !strings.Contains(cmd, want) {
		t.Fatalf("resume command %q does not contain %q", cmd, want)
	}
	if strings.Contains(cmd, "--continue") {
		t.Fatalf("must not use --continue (newest-in-cwd is multi-seat unsafe): %q", cmd)
	}
	// Flag then id, not a glued equals-form, matching grok --resume <SESSION_ID>.
	if strings.Contains(cmd, "--resume="+grokResumeID) {
		t.Fatalf("grok resume is space-separated, got equals-form: %q", cmd)
	}
}

func TestCanRestartGeneric_GrokWithoutResumeFlag(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "")

	inst := &Instance{
		Tool:             "grok",
		Command:          grokBinary,
		GenericSessionID: grokResumeID,
	}
	if inst.CanRestartGeneric() {
		t.Fatal("CanRestartGeneric must be false without resume_flag (the live avicenna misconfig)")
	}
	cmd := inst.buildGenericCommand(inst.Command)
	if strings.Contains(cmd, "--resume") {
		t.Fatalf("without resume_flag the spawn must be a fresh grok, got %q", cmd)
	}
}

func TestBuildGenericCommand_GrokSharedCwdDistinctIDs(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "--resume")

	a := &Instance{
		ID: "admin", Title: "agent-deck-admin", Tool: "grok",
		Command: grokBinary, ProjectPath: "/Users/charles",
		GenericSessionID: grokResumeID,
	}
	b := &Instance{
		ID: "relay", Title: "relay-admin", Tool: "grok",
		Command: grokBinary, ProjectPath: "/Users/charles",
		GenericSessionID: "01a02438-3c1b-7b33-aaff-e214071faffd",
	}
	cmdA := a.buildGenericCommand(a.Command)
	cmdB := b.buildGenericCommand(b.Command)
	if !strings.Contains(cmdA, grokResumeID) {
		t.Fatalf("admin resume missing: %q", cmdA)
	}
	if !strings.Contains(cmdB, b.GenericSessionID) {
		t.Fatalf("relay resume missing: %q", cmdB)
	}
	if strings.Contains(cmdA, b.GenericSessionID) || strings.Contains(cmdB, grokResumeID) {
		t.Fatalf("shared cwd must not cross-bind\n admin=%q\n relay=%q", cmdA, cmdB)
	}
}
