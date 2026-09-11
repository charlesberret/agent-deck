package ui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Trust egress glyphs — additive to brand ToolIcon. Brand color ≠ trust.
// Canon: ~/cloud/sync/system/agent-deck-trust-registry.yaml
//        ~/cloud/sync/system/trust-register.md (§ agent-deck glance badge)
//
// Row shows a single egress mark before the brand icon:
//
//	⌂ local/mesh · ↑ vendor (data leaves) · ! shell/unbounded
//
// Prefer single-cell symbols (☁ was muddy/double-width in Ghostty).
// Full multi-factor badge (L:c:ip) stays in `agent-deck-trust` CLI / detail
// panel later — this is the v1 glance the UX brief called for.

const (
	egressLocal  = "⌂"
	egressVendor = "↑"
	egressAlert  = "!"
)

// builtinEgress is the fallback when the fleet registry file is missing
// (e.g. CI machines without the Syncthing tree). Keep aligned with
// system/agent-deck-trust-registry.yaml tools: section.
var builtinEgress = map[string]string{
	"ornith":         egressLocal,
	"penlight":       egressLocal,
	"local-ornith":   egressLocal,
	"local-penlight": egressLocal,
	// Fleet oll-* grades are Ollama Cloud (*:cloud) — vendor egress.
	"oll-flash":      egressVendor,
	"oll-gemma":      egressVendor,
	"oll-light":      egressVendor,
	"oll-mid":        egressVendor,
	"oll-heavy":      egressVendor,
	"oll-hard-heavy": egressVendor,
	"oll-pro":        egressVendor,
	"oll-vision":     egressVendor,
	"oll-nemotron":   egressVendor,
	"oll-oss":        egressVendor,
	"oll-memo":       egressVendor,
	"wk-kimi-code":   egressVendor,
	"wk-glm":         egressVendor,
	"wk-m3":          egressVendor,
	"wk-granite":     egressLocal, // local granite persona — not cloud
	"claude":         egressVendor,
	"grok":           egressVendor,
	"gemini":         egressVendor,
	"agy":            egressVendor,
	"codex":          egressVendor,
	"copilot":        egressVendor,
	"cursor":         egressVendor,
	"shell":          egressAlert,
}

type trustRegistryFile struct {
	Tools map[string]struct {
		Glyph string `yaml:"glyph"`
	} `yaml:"tools"`
}

var (
	egressOnce sync.Once
	egressMap  map[string]string
)

func trustRegistryPath() string {
	if p := os.Getenv("AGENT_DECK_TRUST_REGISTRY"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "cloud", "sync", "system", "agent-deck-trust-registry.yaml")
}

func loadEgressMap() map[string]string {
	out := make(map[string]string, len(builtinEgress))
	for k, v := range builtinEgress {
		out[k] = v
	}
	path := trustRegistryPath()
	if path == "" {
		return out
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var doc trustRegistryFile
	if err := yaml.Unmarshal(data, &doc); err != nil || doc.Tools == nil {
		return out
	}
	for name, row := range doc.Tools {
		g := strings.TrimSpace(row.Glyph)
		if g == "" {
			continue
		}
		out[name] = g
	}
	return out
}

func egressTable() map[string]string {
	egressOnce.Do(func() {
		egressMap = loadEgressMap()
	})
	return egressMap
}

// TrustEgressGlyph returns the compact egress mark for a deck tool id, or "".
func TrustEgressGlyph(tool string) string {
	tool = strings.TrimSpace(strings.ToLower(tool))
	if tool == "" {
		return ""
	}
	if g, ok := egressTable()[tool]; ok {
		return g
	}
	// Heuristics for unregistered custom tools
	switch {
	case strings.HasPrefix(tool, "wk-"):
		return egressVendor
	case strings.HasPrefix(tool, "bt-"):
		return egressVendor // braintrust amber seats are cloud
	case strings.HasPrefix(tool, "local-"):
		return egressLocal
	case strings.HasPrefix(tool, "oll-"):
		// most oll-* on this fleet are Ollama Cloud grades
		return egressVendor
	default:
		return ""
	}
}

// resetTrustEgressForTest clears the cache (tests only).
func resetTrustEgressForTest() {
	egressOnce = sync.Once{}
	egressMap = nil
}
