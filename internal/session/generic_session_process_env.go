package session

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// lookupProcessEnv reads one environment variable from pid. Tests swap this.
var lookupProcessEnv = lookupProcessEnvDefault

// lookupProcessEnvDefault prefers Linux procfs, then `ps eww` (macOS / BSD).
func lookupProcessEnvDefault(pid int, key string) string {
	if pid <= 0 || key == "" {
		return ""
	}
	if v := lookupProcEnviron(pid, key); v != "" {
		return v
	}
	return lookupPSEwwEnv(pid, key)
}

func lookupProcEnviron(pid int, key string) string {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return ""
	}
	for _, entry := range strings.Split(string(raw), "\x00") {
		k, v, ok := strings.Cut(entry, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}

func lookupPSEwwEnv(pid int, key string) string {
	// #nosec G204 -- pid is strconv.Itoa of an int; key is not interpolated.
	out, err := exec.Command("ps", "eww", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return ""
	}
	return parsePSEwwEnv(string(out), key)
}

// parsePSEwwEnv pulls KEY=value from `ps eww` output. The command and env
// share one line; values are whitespace-delimited tokens.
func parsePSEwwEnv(output, key string) string {
	if key == "" {
		return ""
	}
	prefix := key + "="
	for _, tok := range strings.Fields(output) {
		if strings.HasPrefix(tok, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(tok, prefix))
		}
	}
	return ""
}

// uniqueNonEmptyStrings returns the sole distinct non-empty value, or "" if
// none or more than one. Multiple values are leftover MCP children from a
// previous conversation in the same pane — guessing would rebind the seat
// to the wrong chat.
func uniqueNonEmptyStrings(in []string) string {
	seen := make(map[string]struct{}, len(in))
	first := ""
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		if first == "" {
			first = s
		}
	}
	if len(seen) != 1 {
		return ""
	}
	return first
}

// genericSessionIDFromProcessTree reads session_id_env from pane descendants.
// Grok (and similar TUIs) export the id on hook/MCP children, not in tmux
// show-environment, so the persist-PR write-through never fired.
//
// Only used when no id is already persisted: a live pane can also contain
// leftover children from an older conversation, and those must not clobber
// a binding we already trust.
func (i *Instance) genericSessionIDFromProcessTree() string {
	if i == nil {
		return ""
	}
	toolDef := GetToolDef(i.Tool)
	if toolDef == nil || strings.TrimSpace(toolDef.SessionIDEnv) == "" {
		return ""
	}
	return i.genericSessionIDFromPIDs(nil) // nil means "collect from tmux"
}

// genericSessionIDFromPIDs is the testable core. A nil/empty pid list means
// collect the live pane tree; tests pass an explicit set and stub lookupProcessEnv.
func (i *Instance) genericSessionIDFromPIDs(pids []int) string {
	toolDef := GetToolDef(i.Tool)
	if toolDef == nil {
		return ""
	}
	envName := strings.TrimSpace(toolDef.SessionIDEnv)
	if envName == "" {
		return ""
	}
	if len(pids) == 0 {
		var err error
		pids, err = i.collectTmuxPaneProcessTreePIDs()
		if len(pids) == 0 {
			_ = err
			return ""
		}
	}
	found := make([]string, 0, len(pids))
	for _, pid := range pids {
		if v := strings.TrimSpace(lookupProcessEnv(pid, envName)); v != "" {
			found = append(found, v)
		}
	}
	return uniqueNonEmptyStrings(found)
}

func (i *Instance) publishGenericSessionIDToTmux(id string) {
	if i == nil || i.tmuxSession == nil || strings.TrimSpace(id) == "" {
		return
	}
	for _, name := range i.genericSessionEnvNames() {
		_ = i.tmuxSession.SetEnvironment(name, id)
	}
}

// captureGenericSessionIDFromProcessTree is first-run learn: persist + publish
// when the pane has exactly one live conversation id and we have none yet.
func (i *Instance) captureGenericSessionIDFromProcessTree() string {
	id := strings.TrimSpace(i.genericSessionIDFromProcessTree())
	if id == "" {
		return ""
	}
	i.persistGenericSessionIDIfChanged(id)
	i.publishGenericSessionIDToTmux(id)
	return id
}
