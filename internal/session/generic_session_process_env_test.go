package session

import (
	"testing"
)

func TestParsePSEwwEnv_GrokChild(t *testing.T) {
	// Shape observed on avicenna: grok itself has no GROK_SESSION_ID; MCP
	// children inherit it and `ps eww` dumps command + env on one line.
	out := `/opt/homebrew/Cellar/python@3.14/3.14.7/Resources/Python.app/Contents/MacOS/Python /Users/charles/cloud/git-projects/t PWD=/Users/charles GROK_SESSION_ID=019fe0ec-7a6d-7632-bfc8-616d13f6f181 PATH=/opt/homebrew/bin`
	got := parsePSEwwEnv(out, "GROK_SESSION_ID")
	want := "019fe0ec-7a6d-7632-bfc8-616d13f6f181"
	if got != want {
		t.Fatalf("parsePSEwwEnv = %q, want %q", got, want)
	}
	if parsePSEwwEnv(out, "MISSING") != "" {
		t.Fatal("missing key must be empty")
	}
	if parsePSEwwEnv("grok", "GROK_SESSION_ID") != "" {
		t.Fatal("grok binary line with no env must be empty")
	}
}

func TestUniqueNonEmptyStrings(t *testing.T) {
	if got := uniqueNonEmptyStrings(nil); got != "" {
		t.Fatalf("nil = %q", got)
	}
	if got := uniqueNonEmptyStrings([]string{"", "  "}); got != "" {
		t.Fatalf("blanks = %q", got)
	}
	if got := uniqueNonEmptyStrings([]string{"a", "a", " a "}); got != "a" {
		t.Fatalf("dedupe = %q, want a", got)
	}
	if got := uniqueNonEmptyStrings([]string{"old", "fresh"}); got != "" {
		t.Fatalf("conflict must refuse, got %q", got)
	}
}

func TestGenericSessionIDFromPIDs_UniqueChild(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "--resume")

	orig := lookupProcessEnv
	t.Cleanup(func() { lookupProcessEnv = orig })
	lookupProcessEnv = func(pid int, key string) string {
		if key != "GROK_SESSION_ID" {
			return ""
		}
		switch pid {
		case 10: // grok binary
			return ""
		case 11, 12: // MCP children
			return "019fe0ec-7a6d-7632-bfc8-616d13f6f181"
		default:
			return ""
		}
	}

	inst := &Instance{Tool: "grok", Command: "/Users/charles/.local/bin/grok"}
	got := inst.genericSessionIDFromPIDs([]int{10, 11, 12})
	want := "019fe0ec-7a6d-7632-bfc8-616d13f6f181"
	if got != want {
		t.Fatalf("from PIDs = %q, want %q", got, want)
	}
}

func TestGenericSessionIDFromPIDs_ConflictRefuses(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "--resume")

	orig := lookupProcessEnv
	t.Cleanup(func() { lookupProcessEnv = orig })
	lookupProcessEnv = func(pid int, key string) string {
		if key != "GROK_SESSION_ID" {
			return ""
		}
		if pid == 1 {
			return "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		}
		return "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	}

	inst := &Instance{Tool: "grok"}
	if got := inst.genericSessionIDFromPIDs([]int{1, 2}); got != "" {
		t.Fatalf("conflict must be empty, got %q", got)
	}
}

func TestGenericSessionIDFromPIDs_NoSessionIDEnv(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "") // no session_id_env

	orig := lookupProcessEnv
	t.Cleanup(func() { lookupProcessEnv = orig })
	called := false
	lookupProcessEnv = func(pid int, key string) string {
		called = true
		return "should-not-run"
	}
	inst := &Instance{Tool: "grok"}
	if got := inst.genericSessionIDFromPIDs([]int{1}); got != "" {
		t.Fatalf("no session_id_env must skip, got %q", got)
	}
	if called {
		t.Fatal("must not walk processes when session_id_env is unset")
	}
}

func TestGetGenericSessionID_PersistedWinsOverProcessTree(t *testing.T) {
	home := isolateToolConfigHome(t)
	writeGrokToolConfig(t, home, "--resume")

	orig := lookupProcessEnv
	t.Cleanup(func() { lookupProcessEnv = orig })
	lookupProcessEnv = func(pid int, key string) string {
		t.Fatal("process tree must not run when an id is already persisted")
		return "from-tree"
	}

	inst := &Instance{
		Tool:             "grok",
		Command:          "/Users/charles/.local/bin/grok",
		GenericSessionID: "019fe0ec-7a6d-7632-bfc8-616d13f6f181",
		tmuxSession:      nil,
	}
	got := inst.GetGenericSessionID()
	if got != "019fe0ec-7a6d-7632-bfc8-616d13f6f181" {
		t.Fatalf("GetGenericSessionID = %q, want persisted id", got)
	}
}
