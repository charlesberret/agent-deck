package ui

import "testing"

func TestTopLevelGroupPath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"grades", "grades"},
		{"household-agents", "household-agents"},
		{"household-agents/clerk", "household-agents"},
		{"household-agents/clerk/nested", "household-agents"},
		{"remotes/hesiod", "remotes/hesiod"},
		{"remotes/hesiod/grades", "remotes/hesiod"},
		{"remotes/hesiod/household-agents/clerk", "remotes/hesiod"},
		{"xanadu-agents/alchemist", "xanadu-agents"},
	}
	for _, tt := range tests {
		if got := topLevelGroupPath(tt.in); got != tt.want {
			t.Errorf("topLevelGroupPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
