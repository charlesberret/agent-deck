package ui

import (
	"os"
	"path/filepath"
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

// Trust flags rendered after the egress glyph.
const (
	trustFlagOverprivilege = "!"
	trustFlagBlocked       = "×"
)

type subjectRule struct {
	Match string `yaml:"match"`
	Class string `yaml:"class"`
}

type trustSubjectFile struct {
	Surfaces []struct {
		ID        string `yaml:"id"`
		Locus     string `yaml:"provider_locus"`
		Retention string `yaml:"retention"`
		WriteMode string `yaml:"write_mode"`
		Ceiling   string `yaml:"subject_ceiling"`
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

type trustModel struct {
	rules        []subjectRule
	defaultClass SubjectClass
	toolSurface  map[string]surfaceInfo
}

var (
	subjectOnce  sync.Once
	subjectModel *trustModel
)

func loadTrustModel() *trustModel {
	// Fail closed upward: with no registry we still classify unknown work as
	// project-ip rather than treating it as public.
	m := &trustModel{
		defaultClass: SubjectProjectIP,
		toolSurface:  map[string]surfaceInfo{},
	}
	path := trustRegistryPath()
	if path == "" {
		return m
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	var doc trustSubjectFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return m
	}

	if c, ok := subjectNames[strings.TrimSpace(doc.SubjectPaths.Default)]; ok {
		m.defaultClass = c
	}
	m.rules = doc.SubjectPaths.Rules

	surfaces := make(map[string]surfaceInfo, len(doc.Surfaces))
	for _, s := range doc.Surfaces {
		ceiling, ok := subjectNames[strings.TrimSpace(s.Ceiling)]
		if !ok {
			// An unreadable ceiling must not silently become "unlimited".
			ceiling = SubjectProjectIP
		}
		surfaces[s.ID] = surfaceInfo{
			locus:     strings.TrimSpace(s.Locus),
			retention: strings.TrimSpace(s.Retention),
			writeMode: strings.TrimSpace(s.WriteMode),
			ceiling:   ceiling,
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
			si.retention = "vendor-may-retain"
			if si.ceiling > SubjectProjectIP {
				si.ceiling = SubjectProjectIP
			}
		}
		// T2 — retention is a fact of the provider, not of manners. The
		// Claude/Grok/Gemini rows record `unknown` pending ZDR ratification
		// (register gap 7); `unknown` must never gate more weakly than
		// `vendor-may-retain`, or H1 silently switches off for those seats.
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

// tildePath rewrites a leading $HOME to "~" so registry globs stay portable
// across the fleet's differing home paths (darwin /Users, linux /home).
func tildePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// Normalize ".." and doubled separators before matching: a prefix rule
	// must not be dodged by ~/cloud/sync/keys/../keys or ~/cloud//sync.
	// (Instance.ProjectPath is already symlink-resolved upstream.)
	if filepath.IsAbs(p) {
		p = filepath.Clean(p)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if p == home {
			return "~"
		}
		if strings.HasPrefix(p, home+string(filepath.Separator)) {
			return "~" + p[len(home):]
		}
	}
	return p
}

// globMatch supports the two shapes the registry uses: a "prefix/**" subtree
// and a "**/name" basename match. filepath.Match alone handles neither.
func globMatch(pattern, path string) bool {
	switch {
	case strings.HasSuffix(pattern, "/**"):
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	case strings.HasPrefix(pattern, "**/"):
		base := strings.TrimPrefix(pattern, "**/")
		ok, err := filepath.Match(base, filepath.Base(path))
		return err == nil && ok
	default:
		if pattern == path {
			return true
		}
		ok, err := filepath.Match(pattern, path)
		return err == nil && ok
	}
}

// TrustSubjectClass returns the sensitivity class of the work at path.
// First matching rule wins, so registry order is load-bearing; an unmatched
// path fails closed upward to the registry default (project-ip).
func TrustSubjectClass(path string) SubjectClass {
	m := trustModelTable()
	p := tildePath(path)
	if p == "" {
		return m.defaultClass
	}
	for _, r := range m.rules {
		if globMatch(r.Match, p) {
			if c, ok := subjectNames[strings.TrimSpace(r.Class)]; ok {
				return c
			}
		}
	}
	return m.defaultClass
}

// TrustFlag reports whether this tool is appropriate for work at path.
//
//	"×" the provider may not hold this subject at all (register H1/H2)
//	"!" the seat is over-privileged for the subject (register W1)
//	""  nothing to say
//
// Two register warnings are deliberately NOT rendered here, because the row is
// the scannability surface and the operator card's glance test only works if
// "!" and "×" stay rare:
//
//   - W3 (project IP on a vendor seat) — the ☁ glyph already carries it, and it
//     would fire on every cloud coding session.
//   - W1 restricted to `destructive` only. W1 as written also covers `mutate`
//     on <= internal work, but that describes the house's ordinary state (a
//     Claude seat editing system/ is the job, not a finding). W1's "and no
//     grant named for that tree" clause is what would have excluded it, and no
//     grant mechanism exists yet — so rendering it would be a standing false
//     positive. `agent-deck-trust match` still reports the full rule.
func TrustFlag(tool, path string) string {
	tool = strings.TrimSpace(strings.ToLower(tool))
	if tool == "" {
		return ""
	}
	si, ok := trustModelTable().toolSurface[tool]
	if !ok {
		return ""
	}
	subject := TrustSubjectClass(path)

	// H1 — secret material must never sit under a vendor seat.
	if si.retention == "vendor-may-retain" &&
		(subject == SubjectCredential || subject == SubjectSealed) {
		return trustFlagBlocked
	}
	// H2 — subject exceeds what this surface may touch.
	if subject > si.ceiling {
		return trustFlagBlocked
	}
	// W1 (narrowed — see doc comment) — an unscoped destructive seat.
	if si.writeMode == "destructive" {
		return trustFlagOverprivilege
	}
	return ""
}

// resetTrustSubjectForTest clears the cache (tests only).
func resetTrustSubjectForTest() {
	subjectOnce = sync.Once{}
	subjectModel = nil
}
