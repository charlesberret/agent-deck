package session

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// adjectives for auto-generated session names (nature/weather themed)
var adjectives = []string{
	"amber", "ancient", "arctic", "autumn", "azure",
	"blazing", "bold", "bright", "bronze", "calm",
	"cedar", "clear", "coastal", "cool", "coral",
	"cosmic", "crimson", "crystal", "dappled", "dawn",
	"deep", "desert", "drifting", "dusky", "eager",
	"emerald", "fading", "fern", "fierce", "floral",
	"foggy", "forest", "frosty", "gentle", "gilded",
	"glacial", "gleaming", "golden", "granite", "hazy",
	"hidden", "hollow", "hushed", "icy", "indigo",
	"iron", "ivory", "jade", "keen", "lapis",
	"leafy", "light", "lively", "lunar", "marble",
	"meadow", "misty", "molten", "mossy", "nimble",
	"noble", "northern", "obsidian", "opal", "pale",
	"pearly", "pine", "polar", "prairie", "quartz",
	"quiet", "radiant", "rapid", "risen", "rocky",
	"rosy", "ruby", "rustic", "sandy", "scarlet",
	"shadow", "shining", "silent", "silver", "slate",
	"smoky", "solar", "spring", "starry", "steady",
	"stone", "stormy", "sunlit", "swift", "tawny",
	"thorny", "tidal", "topaz", "twilight", "verdant",
	"violet", "vivid", "wandering", "warm", "wild",
	"windy", "woven", "young", "zephyr",
}

// nouns for auto-generated session names (animals/nature themed)
var nouns = []string{
	"badger", "bear", "birch", "bison", "brook",
	"canyon", "cedar", "cliff", "cloud", "condor",
	"coral", "cougar", "crane", "creek", "crow",
	"delta", "dove", "dune", "eagle", "elm",
	"falcon", "fern", "finch", "fjord", "flower",
	"forest", "fox", "frost", "garden", "glacier",
	"grove", "gull", "harbor", "hawk", "heron",
	"hill", "hollow", "horizon", "island", "ivy",
	"jay", "juniper", "lake", "lark", "leaf",
	"lily", "lotus", "lynx", "maple", "marsh",
	"meadow", "mesa", "moon", "moss", "oak",
	"ocean", "orchid", "osprey", "otter", "owl",
	"palm", "panther", "peak", "pebble", "pine",
	"plover", "pond", "quail", "rain", "raven",
	"reef", "ridge", "river", "robin", "sage",
	"salmon", "shore", "sky", "sparrow", "spruce",
	"star", "stone", "storm", "stream", "summit",
	"swallow", "thistle", "thorn", "tide", "trail",
	"tulip", "valley", "vine", "wave", "willow",
	"wolf", "wren", "yarrow", "yew",
}

// GenerateSessionName returns a random "adjective-noun" name.
func GenerateSessionName() string {
	adj := adjectives[cryptoRandInt(len(adjectives))]
	noun := nouns[cryptoRandInt(len(nouns))]
	return adj + "-" + noun
}

// GenerateUniqueSessionName generates a name that doesn't collide with
// existing session titles in the given group. Retries up to 10 times,
// then falls back to appending a timestamp.
func GenerateUniqueSessionName(instances []*Instance, groupPath string) string {
	existing := make(map[string]bool)
	for _, inst := range instances {
		if inst.GroupPath == groupPath {
			existing[inst.Title] = true
		}
	}

	for range 10 {
		name := GenerateSessionName()
		if !existing[name] {
			return name
		}
	}

	// Fallback: append timestamp to guarantee uniqueness
	name := GenerateSessionName()
	return fmt.Sprintf("%s-%d", name, time.Now().Unix())
}

// generatedNameSets is built once from the adjective/noun lists so
// LooksLikeGeneratedSessionName is a pair of map lookups.
var generatedNameSets struct {
	once       sync.Once
	adjectives map[string]bool
	nouns      map[string]bool
}

func initGeneratedNameSets() {
	generatedNameSets.once.Do(func() {
		generatedNameSets.adjectives = make(map[string]bool, len(adjectives))
		for _, a := range adjectives {
			generatedNameSets.adjectives[a] = true
		}
		generatedNameSets.nouns = make(map[string]bool, len(nouns))
		for _, n := range nouns {
			generatedNameSets.nouns[n] = true
		}
	})
}

// LooksLikeGeneratedSessionName reports whether title is an adjective-noun
// handle from GenerateSessionName (or its timestamp-suffixed uniqueness
// fallback). Used to mark leftover unnamed seats that predate the AutoName flag.
func LooksLikeGeneratedSessionName(title string) bool {
	t := strings.TrimSpace(strings.ToLower(title))
	if t == "" {
		return false
	}
	initGeneratedNameSets()
	parts := strings.Split(t, "-")
	if len(parts) < 2 {
		return false
	}
	adj, noun := parts[0], parts[1]
	if !generatedNameSets.adjectives[adj] || !generatedNameSets.nouns[noun] {
		return false
	}
	// Bare "misty-owl" or uniqueness fallback "misty-owl-1718000000".
	if len(parts) == 2 {
		return true
	}
	if len(parts) == 3 {
		for _, r := range parts[2] {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(parts[2]) > 0
	}
	return false
}

// IsContinuityUnnamed is true when the session has no explicit name that
// agent-deck can treat as a stable continuity handle.
//
// AutoName sessions are unnamed by construction (quick-create / TUI-Q).
// A still-generated adjective-noun title that was never title-locked is
// treated the same, so pre-AutoName seats get the marker too. An explicit
// rename (TUI `r`, create dialog, `-t`, or Claude `/rename` which clears
// AutoName) drops the marker.
func IsContinuityUnnamed(autoName, titleLocked bool, title string) bool {
	if autoName {
		return true
	}
	if titleLocked {
		return false
	}
	return LooksLikeGeneratedSessionName(title)
}

// cryptoRandInt returns a cryptographically random int in [0, max).
func cryptoRandInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// Fallback to timestamp-based selection if crypto/rand fails
		return int(time.Now().UnixNano() % int64(max))
	}
	return int(n.Int64())
}
