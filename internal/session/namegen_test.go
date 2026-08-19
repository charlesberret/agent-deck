package session

import (
	"strings"
	"testing"
)

func TestGenerateSessionName(t *testing.T) {
	name := GenerateSessionName()

	// Must be "adjective-noun" format
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("expected adjective-noun format, got %q", name)
	}
	if parts[0] == "" || parts[1] == "" {
		t.Fatalf("empty part in name %q", name)
	}
}

func TestGenerateSessionName_Unique(t *testing.T) {
	seen := make(map[string]bool)
	dupes := 0
	const iterations = 200

	for range iterations {
		name := GenerateSessionName()
		if seen[name] {
			dupes++
		}
		seen[name] = true
	}

	// With ~10,000 combinations and 200 draws, collisions should be rare
	if dupes > 10 {
		t.Errorf("too many duplicates: %d out of %d", dupes, iterations)
	}
}

func TestGenerateUniqueSessionName(t *testing.T) {
	instances := []*Instance{
		{Title: "swift-fox", GroupPath: "work"},
		{Title: "golden-eagle", GroupPath: "work"},
		{Title: "calm-brook", GroupPath: "personal"},
	}

	name := GenerateUniqueSessionName(instances, "work")

	// Must not collide with existing work group sessions
	if name == "swift-fox" || name == "golden-eagle" {
		t.Errorf("generated name %q collides with existing session", name)
	}

	// Must still be valid format
	if !strings.Contains(name, "-") {
		t.Errorf("expected hyphenated name, got %q", name)
	}
}

func TestGenerateUniqueSessionName_DifferentGroup(t *testing.T) {
	instances := []*Instance{
		{Title: "calm-brook", GroupPath: "personal"},
	}

	// "calm-brook" exists in "personal" but we're generating for "work",
	// so it should be allowed (no collision)
	// We can't guarantee the exact name, but it shouldn't error
	name := GenerateUniqueSessionName(instances, "work")
	if name == "" {
		t.Error("expected non-empty name")
	}
}

func TestGenerateUniqueSessionName_EmptyInstances(t *testing.T) {
	name := GenerateUniqueSessionName(nil, "work")
	if name == "" {
		t.Error("expected non-empty name")
	}
	if !strings.Contains(name, "-") {
		t.Errorf("expected hyphenated name, got %q", name)
	}
}

func TestLooksLikeGeneratedSessionName(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{"misty-owl", true},
		{"Misty-Owl", true},
		{"swift-fox", true},
		{"calm-brook-1718000000", true},
		{"clerk", false},
		{"alchemist", false},
		{"kettle-core", false},
		{"misty-owl-extra", false},
		{"notanadj-owl", false},
		{"misty-notanoun", false},
		{"", false},
		{"justone", false},
	}
	for _, tc := range cases {
		if got := LooksLikeGeneratedSessionName(tc.title); got != tc.want {
			t.Errorf("LooksLikeGeneratedSessionName(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
}

func TestIsContinuityUnnamed(t *testing.T) {
	if !IsContinuityUnnamed(true, false, "clerk") {
		t.Error("AutoName must mark even after a display title is promoted")
	}
	if IsContinuityUnnamed(false, true, "misty-owl") {
		t.Error("title-locked generated handle is an explicit name")
	}
	if IsContinuityUnnamed(false, false, "alchemist") {
		t.Error("distinctive unlocked title is a continuity handle")
	}
	if !IsContinuityUnnamed(false, false, "misty-owl") {
		t.Error("unlocked generated handle should still be unnamed")
	}
}

func TestCryptoRandInt(t *testing.T) {
	// Should return values in [0, max)
	for range 100 {
		n := cryptoRandInt(10)
		if n < 0 || n >= 10 {
			t.Fatalf("cryptoRandInt(10) = %d, want [0, 10)", n)
		}
	}
}
