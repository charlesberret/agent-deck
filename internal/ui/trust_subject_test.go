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
		{[]string{"cloud", "sync", "system"}, SubjectPersonal}, // holds contacts.md
		{[]string{"cloud", "sync", "system", "runbooks"}, SubjectInternal},
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
	// system/ holds contacts.md, so a seat rooted there inherits personal.
	sys := homeJoin(t, "cloud", "sync", "system")
	if got := TrustSubjectClass(sys); got != SubjectPersonal {
		t.Errorf("system/ = %v, want personal (contains contacts.md)", got)
	}
	// A system/ subdirectory that holds nothing sensitive stays internal.
	if got := TrustSubjectClass(homeJoin(t, "cloud", "sync", "system", "runbooks")); got != SubjectInternal {
		t.Errorf("system/runbooks = %v, want internal", got)
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
// Containment is COMPUTED, so this holds for containers nobody listed.
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
	runbooks := homeJoin(t, "cloud", "sync", "system", "runbooks")
	if got := TrustFlag("claude", runbooks); got != "" {
		t.Errorf("TrustFlag(claude, system/runbooks) = %q, want no flag (ops work is the job)", got)
	}
}

// W1 is not painted on the row: narrowed to `destructive` it was a synonym
// for "tool == shell", constant across every directory, and the shell row
// already carries "!" from its egress glyph. The mark must vary with the
// session or the eye trains off it.
func TestTrustFlag_ShellNotDoubleMarked(t *testing.T) {
	useFleetRegistry(t)

	if got := TrustFlag("shell", homeJoin(t, "cloud", "sync", "code")); got != "" {
		t.Errorf("TrustFlag(shell, code) = %q, want no flag (egress ! already says unbounded)", got)
	}
	// It is still blocked where the subject genuinely exceeds the ceiling.
	if got := TrustFlag("claude", homeJoin(t, "cloud", "sync", "keys")); got != trustFlagBlocked {
		t.Errorf("claude on keys = %q, want %q", got, trustFlagBlocked)
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

// An unregistered tool must NOT be silent. trust_egress.go still paints ☁ for
// bt-*/wk-*/oll-* by prefix heuristic, so a blank flag would leave the row
// looking evaluated when nothing evaluated it.
func TestTrustFlag_UnknownToolIsMarkedUnclassified(t *testing.T) {
	useFleetRegistry(t)

	if got := TrustFlag("some-unregistered-tool", homeJoin(t, "cloud", "sync", "keys")); got != trustFlagUnclassified {
		t.Errorf("unknown tool = %q, want %q", got, trustFlagUnclassified)
	}
}

func TestRenderToolBadgeForPath_FlagsBlockedPairing(t *testing.T) {
	forceTrueColorProfile()
	useFleetRegistry(t)

	style := GetToolStyle("claude")
	keys := homeJoin(t, "cloud", "sync", "keys")

	got := renderToolBadgeForPath("claude", []string{keys}, style)
	if !strings.Contains(got, trustFlagBlocked) {
		t.Fatalf("expected %q in badge for claude on keys/, got %q", trustFlagBlocked, got)
	}

	repo := homeJoin(t, "cloud", "git-projects", "kettle-core")
	plain := renderToolBadgeForPath("claude", []string{repo}, style)
	if strings.Contains(plain, trustFlagBlocked) || strings.Contains(plain, trustFlagUnclassified) {
		t.Fatalf("ordinary repo work must be unflagged, got %q", plain)
	}
	if plain != renderToolBadge("claude", style) {
		t.Fatalf("unflagged badge must equal the plain badge; got %q", plain)
	}

	// An additional path into a personal tree must taint the whole session:
	// the verdict is the max over every tree the seat can read.
	career := homeJoin(t, "cloud", "sync", "career")
	multi := renderToolBadgeForPath("claude", []string{repo, career}, style)
	if !strings.Contains(multi, trustFlagBlocked) {
		t.Fatalf("repo + career must flag on the worst path, got %q", multi)
	}
}

// Everything the review found reachable-and-silent. Each of these rendered a
// clean badge before, which on a row reads as "checked and clean".
func TestTrustFlag_NoSilentPasses(t *testing.T) {
	useFleetRegistry(t)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}

	cases := []struct {
		name, tool, path, want string
	}{
		// Case-insensitive filesystem: this really resolves to keys/.
		{"mis-cased tree", "claude", filepath.Join(home, "cloud/sync/KEYS"), trustFlagBlocked},
		{"mis-cased home", "claude", "/Users/Charles/cloud/sync/keys", trustFlagBlocked},
		// Remote linux home -- SSH rows carry the remote path.
		{"linux home", "claude", "/home/charles/cloud/sync/keys", trustFlagBlocked},
		{"linux container", "claude", "/home/charles", trustFlagBlocked},
		// Configured deck tools that were absent from the registry.
		{"unregistered bt seat", "bt-oll-heavy", filepath.Join(home, "cloud/sync/notes/interviews"), trustFlagBlocked},
		// Named only in a surface's deck_tool_ids, with no tools: row -- the
		// registry described it all along and it still rendered nothing.
		{"deck_tool_ids-only tool", "codex", filepath.Join(home, "cloud/sync/keys"), trustFlagBlocked},
		{"genuinely absent tool", "some-new-tool", filepath.Join(home, "cloud/sync/keys"), trustFlagUnclassified},
		// Sensitive trees the first pass never listed.
		{"wireguard keys", "claude", filepath.Join(home, ".config/wireguard"), trustFlagBlocked},
		{"mega cold backup", "claude", filepath.Join(home, "MEGA"), trustFlagBlocked},
		{"downloads", "claude", filepath.Join(home, "Downloads"), trustFlagBlocked},
	}
	for _, c := range cases {
		if got := TrustFlag(c.tool, c.path); got != c.want {
			t.Errorf("%s: TrustFlag(%s, %s) = %q, want %q", c.name, c.tool, c.path, got, c.want)
		}
	}
}

// A registry that will not load must not silently switch the control off --
// trust_egress.go keeps painting glyphs from its builtin map, so the row would
// look evaluated while nothing was evaluated.
func TestTrustFlag_UnloadableRegistryIsLoud(t *testing.T) {
	resetTrustSubjectForTest()
	t.Cleanup(resetTrustSubjectForTest)

	bad := filepath.Join(t.TempDir(), "broken.yaml")
	if err := os.WriteFile(bad, []byte("surfaces: [oh: no: yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", bad)

	if got := TrustFlag("claude", "/anywhere"); got != trustFlagUnclassified {
		t.Errorf("broken registry = %q, want %q", got, trustFlagUnclassified)
	}
}

// A typo in `class:` must fail loud, not declassify the tree it names.
func TestTrustSubject_BadClassFailsClosed(t *testing.T) {
	resetTrustSubjectForTest()
	t.Cleanup(resetTrustSubjectForTest)

	reg := filepath.Join(t.TempDir(), "typo.yaml")
	body := `
surfaces:
  - { id: v, provider_locus: vendor, retention: unknown, write_mode: mutate, subject_ceiling: project-ip }
tools:
  claude: { surface: v, egress: vendor, glyph: "x" }
subject_paths:
  default: project-ip
  rules:
    - { match: "~/cloud/sync/keys/**", class: credentials }
`
	if err := os.WriteFile(reg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_DECK_TRUST_REGISTRY", reg)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got := TrustFlag("claude", filepath.Join(home, "cloud/sync/keys")); got == "" {
		t.Error("typo'd class rendered a clean badge on keys/ — must never be silent")
	}
}
