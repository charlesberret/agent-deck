package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Trust subject matching — the *project* half of the register.
//
// trust_egress.go answers "where does this tool send data" from the tool id
// alone. That is provider physics and it never varies by session. This file
// answers the question the operator card actually asks at launch: is this
// provider appropriate *for this project*?
//
// Canon: ~/cloud/sync/system/trust-register.md (§ A subject sensitivity,
//        § Matching rules H1-H7 / W1-W5)
//        ~/cloud/sync/system/agent-deck-trust-registry.yaml (subject_paths)
//
// Governing principle: **silence must mean "checked and clean", never "could
// not tell"**. On a row, an absent mark is read as reassurance, so every path
// that cannot reach a verdict renders "?" instead of nothing.
//
// Deliberately partial: only the rules computable from (tool, cwd) are
// evaluated here. Rules needing a live grant, network mode, or the actual
// read set stay with the host gates (ollama-tool-policy.md) and the operator.

// SubjectClass is register enum A, ordered low -> high.
type SubjectClass int

const (
	SubjectUnknown SubjectClass = iota
	SubjectPublic
	SubjectInternal
	SubjectProjectIP
	SubjectPersonal
	SubjectCredential
	SubjectSealed
)

var subjectNames = map[string]SubjectClass{
	"public":     SubjectPublic,
	"internal":   SubjectInternal,
	"project-ip": SubjectProjectIP,
	"personal":   SubjectPersonal,
	"credential": SubjectCredential,
	"sealed":     SubjectSealed,
}

var subjectShort = map[SubjectClass]string{
	SubjectPublic:     "pub",
	SubjectInternal:   "int",
	SubjectProjectIP:  "ip",
	SubjectPersonal:   "per",
	SubjectCredential: "cred",
	SubjectSealed:     "seal",
}

// Short returns the badge token for a class ("pub", "ip", "seal", ...).
func (c SubjectClass) Short() string { return subjectShort[c] }

// Trust flags rendered as their own space-separated token after the brand icon.
const (
	trustFlagOverprivilege = "!"
	// trustFlagBlocked is circled division slash (⊘) — the monospace-safe
	// "no" sign (circle + diagonal). Not "×"/"✕" (collides with StatusError
	// ✕ before the title) and not "🛇"/"🚫" (emoji presentation; Ghostty
	// often substitutes a red hexagon + ?). The row paints one rune for
	// any hard block (register H1 or H2). CLI match still prints H1/H2;
	// the suffix is not operator-facing until those rules refuse launch.
	trustFlagBlocked = "⊘"
	// trustFlagUnclassified marks a pairing the register could not judge --
	// an unregistered tool, or a registry that would not load. It is louder
	// than nothing on purpose: "unknown" is worse than "over-privileged",
	// and rendering nothing would let the row read as cleared (T4).
	trustFlagUnclassified = "?"
)

type subjectRule struct {
	Match string `yaml:"match"`
	Class string `yaml:"class"`
}

type trustSubjectFile struct {
	Surfaces []struct {
		ID          string   `yaml:"id"`
		Locus       string   `yaml:"provider_locus"`
		Retention   string   `yaml:"retention"`
		WriteMode   string   `yaml:"write_mode"`
		Ceiling     string   `yaml:"subject_ceiling"`
		DeckToolIDs []string `yaml:"deck_tool_ids"`
	} `yaml:"surfaces"`
	Tools map[string]struct {
		Surface string `yaml:"surface"`
		Egress  string `yaml:"egress"`
	} `yaml:"tools"`
	SubjectPaths struct {
		Default string        `yaml:"default"`
		Rules   []subjectRule `yaml:"rules"`
	} `yaml:"subject_paths"`
}

type surfaceInfo struct {
	locus     string
	retention string
	writeMode string
	ceiling   SubjectClass
}

// compiledRule holds a rule as a normalized path prefix. Registry order is no
// longer load-bearing: matching is longest-prefix plus containment, so a rule
// cannot be shadowed by an earlier broad one (the old first-match-wins list
// could be silently declassified by inserting `~/cloud/sync/**` at the top).
type compiledRule struct {
	prefix   string // normalized, case-folded, no trailing "/**"
	class    SubjectClass
	basename string // set for "**/name" rules, which match a cwd's own name
}

type trustModel struct {
	rules        []compiledRule
	defaultClass SubjectClass
	toolSurface  map[string]surfaceInfo
	// degraded is set when the registry could not be read or parsed, or a
	// rule carried an unreadable class. Every verdict then renders "?".
	degraded bool
}

var (
	subjectOnce  sync.Once
	subjectModel *trustModel
)

func loadTrustModel() *trustModel {
	// Fail closed upward: with no registry we still classify unknown work as
	// project-ip rather than treating it as public -- but `degraded` means we
	// say so on the row instead of quietly rendering a clean badge.
	m := &trustModel{
		defaultClass: SubjectProjectIP,
		toolSurface:  map[string]surfaceInfo{},
	}
	path := trustRegistryPath()
	if path == "" {
		m.degraded = true
		return m
	}
	data, err := os.ReadFile(path)
	if err != nil {
		m.degraded = true
		return m
	}
	var doc trustSubjectFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// yaml.Unmarshal fails whole-document, and this file lives on the
		// Syncthing mesh -- a sync conflict or a broken edit on any host
		// would otherwise turn the whole control off with no signal.
		m.degraded = true
		return m
	}

	if c, ok := subjectNames[strings.TrimSpace(doc.SubjectPaths.Default)]; ok {
		m.defaultClass = c
	}

	for _, r := range doc.SubjectPaths.Rules {
		pat := strings.TrimSpace(r.Match)
		if pat == "" {
			continue
		}
		class, ok := subjectNames[strings.TrimSpace(r.Class)]
		if !ok {
			// A typo in `class:` must never declassify. Treat the rule as
			// maximally sensitive and flag the model degraded so the row
			// shows "?" rather than a confidently wrong clean badge.
			class = SubjectSealed
			m.degraded = true
		}
		if strings.HasPrefix(pat, "**/") {
			m.rules = append(m.rules, compiledRule{
				basename: foldPath(strings.TrimPrefix(pat, "**/")),
				class:    class,
			})
			continue
		}
		m.rules = append(m.rules, compiledRule{
			prefix: foldPath(strings.TrimSuffix(pat, "/**")),
			class:  class,
		})
	}

	surfaces := make(map[string]surfaceInfo, len(doc.Surfaces))
	for _, s := range doc.Surfaces {
		ceiling, ok := subjectNames[strings.TrimSpace(s.Ceiling)]
		if !ok {
			// An unreadable ceiling must not silently become "unlimited".
			ceiling = SubjectProjectIP
			m.degraded = true
		}
		surfaces[s.ID] = surfaceInfo{
			locus:     strings.TrimSpace(s.Locus),
			retention: strings.TrimSpace(s.Retention),
			writeMode: strings.TrimSpace(s.WriteMode),
			ceiling:   ceiling,
		}
	}
	// Seed from each surface's deck_tool_ids first. codex/copilot/cursor and
	// friends are already named there but have no `tools:` row, and without
	// this they render "?" despite the file describing them exactly. An
	// explicit tools: row (below) still wins, since it can override egress.
	for _, s := range doc.Surfaces {
		si, ok := surfaces[s.ID]
		if !ok {
			continue
		}
		if si.locus == "vendor" {
			si.retention = "vendor-may-retain"
		}
		for _, id := range s.DeckToolIDs {
			id = strings.ToLower(strings.TrimSpace(id))
			if id != "" {
				m.toolSurface[id] = si
			}
		}
	}

	for name, row := range doc.Tools {
		si, ok := surfaces[row.Surface]
		if !ok {
			continue
		}
		// A tool row's egress overrides a dual surface template: the fleet's
		// oll-* grades launch :cloud tags under a nominally "local" surface,
		// so the tool row is the truthful locus. Mirrors agent-deck-trust.
		if strings.TrimSpace(row.Egress) == "vendor" {
			si.locus = "vendor"
			if si.ceiling > SubjectProjectIP {
				si.ceiling = SubjectProjectIP
			}
		}
		// T2 -- retention is a fact of the provider, not of manners. P0a
		// pinned claude/grok/gemini vendor-may-retain; a leftover unknown
		// on a vendor row must still never gate more weakly than that.
		if si.locus == "vendor" {
			si.retention = "vendor-may-retain"
		}
		m.toolSurface[strings.ToLower(name)] = si
	}
	return m
}

func trustModelTable() *trustModel {
	subjectOnce.Do(func() { subjectModel = loadTrustModel() })
	return subjectModel
}

// homeLike matches any fleet home directory, not just this host's. Every box
// except avicenna/ficino is /home/<user>, and SSH rows carry the *remote*
// path -- rewriting only os.UserHomeDir() left every remote path unmatched,
// which made the SSH row the weakest classifier in the system.
var homeLike = regexp.MustCompile(`^/(?:Users|home)/[^/]+`)

// normalizePath puts a path into the tilde-rooted, case-folded, symlink-free
// form the rules are compiled to.
func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		// Resolve symlinks before matching: a link named outside the map that
		// points into a sensitive tree would otherwise classify by the link.
		// Only works for local existing paths; remote/absent paths fall back
		// to a lexical clean (same pattern as Instance.hookCwdIsForeign).
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			p = resolved
		}
		p = filepath.Clean(p)
	}
	// Compare against both the raw and symlink-resolved home. EvalSymlinks
	// above is a no-op for a path that does not exist (a stale ProjectPath, a
	// remote path), so resolving only the home would leave /var/... facing a
	// resolved /private/var/... and match nothing.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		homes := []string{filepath.Clean(home)}
		if resolved, err := filepath.EvalSymlinks(home); err == nil {
			resolved = filepath.Clean(resolved)
			if resolved != homes[0] {
				homes = append(homes, resolved)
			}
		}
		for _, h := range homes {
			if p == h {
				return "~"
			}
			if strings.HasPrefix(p, h+string(filepath.Separator)) {
				return foldPath("~" + p[len(h):])
			}
		}
	}
	// Remote or foreign home (/home/charles on the linux boxes).
	if loc := homeLike.FindString(p); loc != "" {
		return foldPath("~" + p[len(loc):])
	}
	return foldPath(p)
}

// foldPath case-folds for matching. APFS here is case-insensitive, so
// `cd ~/cloud/sync/KEYS` succeeds and pwd reports the as-typed casing; a
// case-sensitive prefix compare was the cheapest bypass in the set and needed
// no intent to trip. No fleet path depends on case to distinguish two trees.
func foldPath(p string) string { return strings.ToLower(p) }

// TrustSubjectClass returns the sensitivity class of the work at path.
//
// Two things combine, per register § A ("take the max class among trees the
// session may read"):
//
//   - the most specific rule at or above the path (longest prefix wins), and
//   - the highest class of any rule *beneath* it, because a seat rooted at a
//     container can read everything under it.
//
// The second is why ~ classifies sealed. It is computed, not enumerated: a
// hand-written container list needed every new sensitive rule to have all its
// ancestors added by hand, and silently broke the invariant when one was
// forgotten.
func TrustSubjectClass(path string) SubjectClass {
	m := trustModelTable()
	p := normalizePath(path)
	if p == "" {
		return m.defaultClass
	}

	base := m.defaultClass
	bestLen := -1
	contained := SubjectUnknown

	for _, r := range m.rules {
		if r.basename != "" {
			if r.basename == filepath.Base(p) && r.class > contained {
				contained = r.class
			}
			continue
		}
		switch {
		case r.prefix == p || strings.HasPrefix(p, r.prefix+"/"):
			// Rule is at or above the path -- most specific one wins.
			if len(r.prefix) > bestLen {
				bestLen = len(r.prefix)
				base = r.class
			}
		case strings.HasPrefix(r.prefix, p+"/") || p == "~" && strings.HasPrefix(r.prefix, "~/"):
			// Rule is beneath the path -- the seat can read it.
			if r.class > contained {
				contained = r.class
			}
		}
	}
	if contained > base {
		return contained
	}
	return base
}

// TrustFlag reports whether this tool is appropriate for work at path.
//
//	"⊘"  hard-blocked (register H1 or H2). One rune; the H1/H2 split is CLI.
//	"?"  could not judge -- unregistered tool, or registry unreadable
//	""   nothing to say
//
// Two register warnings are deliberately NOT rendered here, because the row is
// the scannability surface and the operator card's glance test only works if
// the marks stay rare:
//
//   - W3 (project IP on a vendor seat) — the ☁ glyph already carries it, and
//     it would fire on every cloud coding session.
//   - W1 entirely. Narrowing it to `destructive` made it a synonym for
//     "tool == shell" (the only destructive surface) — constant across every
//     directory, carrying no session-specific information, and doubling the
//     "!" the shell row already gets from its egress glyph. W1 on `mutate`
//     is worse: it describes the house's ordinary state, and its "no grant
//     named for that tree" clause has no mechanism yet.
//     `agent-deck-trust match` still reports W1 in full.
func TrustFlag(tool, path string) string {
	tool = strings.TrimSpace(strings.ToLower(tool))
	if tool == "" {
		return ""
	}
	m := trustModelTable()
	if m.degraded {
		return trustFlagUnclassified
	}
	si, ok := m.toolSurface[tool]
	if !ok {
		// Unregistered tool. trust_egress.go still paints ☁ for bt-*/wk-*/
		// oll-* by prefix heuristic, so rendering nothing here would leave
		// the row looking evaluated when it was not.
		return trustFlagUnclassified
	}
	subject := TrustSubjectClass(path)

	// H1 — secret material must never sit under a vendor seat.
	// H2 — subject exceeds what this surface may touch.
	// Row does not distinguish; CLI match still names the rule.
	if si.retention == "vendor-may-retain" &&
		(subject == SubjectCredential || subject == SubjectSealed) {
		return trustFlagBlocked
	}
	if subject > si.ceiling {
		return trustFlagBlocked
	}
	return ""
}

// TrustFlagForPaths is TrustFlag over every tree a session can reach, taking
// the worst verdict (register § A: max class over readable trees). An empty
// list is unjudgeable, not clean.
func TrustFlagForPaths(tool string, paths []string) string {
	if len(paths) == 0 {
		return trustFlagUnclassified
	}
	worst := ""
	rank := map[string]int{
		"": 0, trustFlagUnclassified: 1, trustFlagOverprivilege: 2,
		trustFlagBlocked: 3,
	}
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if f := TrustFlag(tool, p); rank[f] > rank[worst] {
			worst = f
		}
	}
	return worst
}

// resetTrustSubjectForTest clears the cache (tests only).
func resetTrustSubjectForTest() {
	subjectOnce = sync.Once{}
	subjectModel = nil
}
