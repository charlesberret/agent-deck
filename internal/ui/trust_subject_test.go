package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// useFleetRegistry points the trust loaders at the real fleet registry so the
// tests exercise the shipped classification, not a fixture that could drift
// away from it. Located relative to this source file, not via $HOME: the suite
// isolates HOME (testutil.IsolateHome), so a home-relative lookup would always
// miss and every assertion below would silently skip.
// Skips when the Syncthing tree is absent (CI machines).
func useFleetRegistry(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("cannot locate test source")
	}
	// .../cloud/git-sources/agent-deck/internal/ui/x_test.go -> .../cloud
	cloudDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	reg := filepath.Join(cloudDir, "sync", "system", "agent-deck-trust-registry.yaml")
	if _, err := os.Stat(reg); err != nil {
		t.Skip("fleet trust registry not present")
	}
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", reg)
	resetTrustSubjectForTest()
	resetTrustEgressForTest()
	t.Cleanup(func() {
		resetTrustSubjectForTest()
		resetTrustEgressForTest()
	})
}

func homeJoin(t *testing.T, parts ...string) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	return filepath.Join(append([]string{home}, parts...)...)
}

func TestTrustSubjectClass_FleetPaths(t *testing.T) {
	useFleetRegistry(t)

	cases := []struct {
		parts []string
		want  SubjectClass
	}{
		{[]string{"cloud", "sync", "notes", "interviews", "hughes"}, SubjectSealed},
		{[]string{"Documents", "OfficialDocs"}, SubjectSealed},
		{[]string{"cloud", "sync", "keys"}, SubjectCredential},
		{[]string{".ssh"}, SubjectCredential},
		{[]string{"cloud", "sync", "life", "health"}, SubjectPersonal},
		{[]string{"cloud", "sync", "career"}, SubjectPersonal},
		{[]string{"cloud", "git-projects", "kettle-core"}, SubjectProjectIP},
		{[]string{"cloud", "sync", "projects", "Active"}, SubjectProjectIP},
		{[]string{"cloud", "sync", "system"}, SubjectInternal},
		{[]string{"cloud", "sync", "metawork"}, SubjectInternal},
		{[]string{"cloud", "sync", "code", "bin"}, SubjectInternal},
	}
	for _, c := range cases {
		p := homeJoin(t, c.parts...)
		if got := TrustSubjectClass(p); got != c.want {
			t.Errorf("TrustSubjectClass(%s) = %v, want %v", p, got, c.want)
		}
	}
}

// Registry order is load-bearing: the specific rules must out-rank the broad
// tree rules that follow them. A regression here silently declassifies the two
// most sensitive trees in the house.
func TestTrustSubjectClass_SpecificRulesBeatBroadOnes(t *testing.T) {
	useFleetRegistry(t)

	contacts := homeJoin(t, "cloud", "sync", "system", "contacts.md")
	if got := TrustSubjectClass(contacts); got != SubjectPersonal {
		t.Errorf("contacts.md = %v, want personal (system/** must not shadow it)", got)
	}
	sys := homeJoin(t, "cloud", "sync", "system")
	if got := TrustSubjectClass(sys); got != SubjectInternal {
		t.Errorf("system/ = %v, want internal", got)
	}

	interviews := homeJoin(t, "cloud", "sync", "notes", "interviews")
	if got := TrustSubjectClass(interviews); got != SubjectSealed {
		t.Errorf("notes/interviews = %v, want sealed (notes/** must not shadow it)", got)
	}
	// notes/ itself contains interviews/, so a seat rooted there inherits the
	// max; an ordinary subdirectory does not.
	notes := homeJoin(t, "cloud", "sync", "notes")
	if got := TrustSubjectClass(notes); got != SubjectSealed {
		t.Errorf("notes/ = %v, want sealed (it contains interviews/)", got)
	}
	zettel := homeJoin(t, "cloud", "sync", "notes", "zettel")
	if got := TrustSubjectClass(zettel); got != SubjectProjectIP {
		t.Errorf("notes/zettel = %v, want project-ip", got)
	}
}

// Register A: a seat takes the max class of the trees it can read. A container
// cwd must not read as safer than what sits beneath it -- ~ reaches keys/ and
// OfficialDocs, and 5 live vendor sessions sat there unmarked before this.
func TestTrustSubjectClass_ContainersInheritMax(t *testing.T) {
	useFleetRegistry(t)

	for _, parts := range [][]string{
		{},                // ~
		{"cloud"},         // ~/cloud
		{"cloud", "sync"}, // ~/cloud/sync
		{"Documents"},     // ~/Documents
	} {
		p := homeJoin(t, parts...)
		if got := TrustSubjectClass(p); got != SubjectSealed {
			t.Errorf("container %s = %v, want sealed", p, got)
		}
		if got := TrustFlag("claude", p); got != trustFlagBlocked {
			t.Errorf("TrustFlag(claude, %s) = %q, want %q", p, got, trustFlagBlocked)
		}
	}
}

// Unknown paths fail closed *upward* — never public.
func TestTrustSubjectClass_UnknownFailsClosed(t *testing.T) {
	useFleetRegistry(t)

	for _, p := range []string{"/var/tmp/whatever", "", homeJoin(t, "Desktop")} {
		got := TrustSubjectClass(p)
		if got < SubjectProjectIP {
			t.Errorf("TrustSubjectClass(%q) = %v; unknown must fail closed upward", p, got)
		}
	}
}

func TestTrustFlag_VendorOnSecretsIsBlocked(t *testing.T) {
	useFleetRegistry(t)

	keys := homeJoin(t, "cloud", "sync", "keys")
	interviews := homeJoin(t, "cloud", "sync", "notes", "interviews", "zimmermann")

	for _, tool := range []string{"claude", "grok", "wk-kimi-code", "oll-heavy"} {
		if got := TrustFlag(tool, keys); got != trustFlagBlocked {
			t.Errorf("TrustFlag(%s, keys) = %q, want %q", tool, got, trustFlagBlocked)
		}
		if got := TrustFlag(tool, interviews); got != trustFlagBlocked {
			t.Errorf("TrustFlag(%s, interviews) = %q, want %q", tool, got, trustFlagBlocked)
		}
	}
}

// The register's default vendor ceiling is project-ip, so personal trees
// (life/, career/) exceed it — that is the "wrong provider for this project"
// signal the badge exists to show.
func TestTrustFlag_VendorOnPersonalIsBlocked(t *testing.T) {
	useFleetRegistry(t)

	career := homeJoin(t, "cloud", "sync", "career")
	if got := TrustFlag("claude", career); got != trustFlagBlocked {
		t.Errorf("TrustFlag(claude, career) = %q, want %q", got, trustFlagBlocked)
	}
	// A local seat may hold personal work — that is the whole point of ornith.
	if got := TrustFlag("ornith", career); got != "" {
		t.Errorf("TrustFlag(ornith, career) = %q, want no flag", got)
	}
}

// The badge is the scannability surface: the two most common session shapes in
// the house must stay unflagged, or "!" stops meaning anything. W3 is carried
// by the ☁ glyph; W1-on-mutate is the house's ordinary working state.
func TestTrustFlag_OrdinaryWorkIsUnflagged(t *testing.T) {
	useFleetRegistry(t)

	repo := homeJoin(t, "cloud", "git-projects", "kettle-core")
	if got := TrustFlag("claude", repo); got != "" {
		t.Errorf("TrustFlag(claude, git-project) = %q, want no flag (W3 rides the ☁)", got)
	}
	sys := homeJoin(t, "cloud", "sync", "system")
	if got := TrustFlag("claude", sys); got != "" {
		t.Errorf("TrustFlag(claude, system) = %q, want no flag (editing system/ is the job)", got)
	}
}

// The unscoped shell seat is the one standing over-privilege the row should
// keep showing — register: "always ! until scoped".
func TestTrustFlag_UnscopedShellIsFlagged(t *testing.T) {
	useFleetRegistry(t)

	if got := TrustFlag("shell", homeJoin(t, "cloud", "sync", "system")); got != trustFlagOverprivilege {
		t.Errorf("TrustFlag(shell, system) = %q, want %q", got, trustFlagOverprivilege)
	}
}

// T2 regression: the claude/grok/gemini rows record retention `unknown`
// (register gap 7, ZDR unratified). H1 must still fire for them -- testing the
// recorded string alone would switch the secrets gate off on exactly the seats
// Charles uses most.
func TestTrustFlag_UnknownRetentionStillGatesAsVendor(t *testing.T) {
	useFleetRegistry(t)

	m := trustModelTable()
	for _, tool := range []string{"claude", "grok", "gemini"} {
		si, ok := m.toolSurface[tool]
		if !ok {
			t.Fatalf("%s missing from trust model", tool)
		}
		if si.retention != "vendor-may-retain" {
			t.Errorf("%s retention = %q, want vendor-may-retain (T2)", tool, si.retention)
		}
	}
}

func TestTrustFlag_UnknownToolIsSilent(t *testing.T) {
	useFleetRegistry(t)

	if got := TrustFlag("some-unregistered-tool", homeJoin(t, "cloud", "sync", "keys")); got != "" {
		t.Errorf("unknown tool = %q, want silent (no false assurance either way)", got)
	}
}

func TestRenderToolBadgeForPath_FlagsBlockedPairing(t *testing.T) {
	forceTrueColorProfile()
	useFleetRegistry(t)

	style := GetToolStyle("claude")
	keys := homeJoin(t, "cloud", "sync", "keys")

	got := renderToolBadgeForPath("claude", keys, style)
	if !strings.Contains(got, trustFlagBlocked) {
		t.Fatalf("expected %q in badge for claude on keys/, got %q", trustFlagBlocked, got)
	}

	repo := homeJoin(t, "cloud", "git-projects", "kettle-core")
	plain := renderToolBadgeForPath("claude", repo, style)
	if strings.Contains(plain, trustFlagBlocked) || strings.Contains(plain, trustFlagOverprivilege) {
		t.Fatalf("ordinary repo work must be unflagged, got %q", plain)
	}
	if plain != renderToolBadge("claude", style) {
		t.Fatalf("unflagged badge must equal the plain badge; got %q", plain)
	}
}

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"~/cloud/sync/keys/**", "~/cloud/sync/keys", true},
		{"~/cloud/sync/keys/**", "~/cloud/sync/keys/wg.conf", true},
		{"~/cloud/sync/keys/**", "~/cloud/sync/keysmith", false},
		{"**/.env", "~/repo/.env", true},
		{"**/.env", "~/repo/env", false},
		{"**/*.pem", "~/a/b/cert.pem", true},
		{"~/cloud/sync/system/contacts.md", "~/cloud/sync/system/contacts.md", true},
	}
	for _, c := range cases {
		if got := globMatch(c.pattern, c.path); got != c.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}
