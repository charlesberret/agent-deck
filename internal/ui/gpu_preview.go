package ui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gpuBoardTTL = 15 * time.Second

type gpuBoardFetchedMsg struct {
	tag    string
	line   string
	failed bool
}

func gpuTagForTool(tool string) string {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "local-ornith", "ornith":
		return "ornith:9b"
	case "local-ornith-35b", "ornith-35":
		return "ornith:35b"
	case "local-gemma4", "gemma4":
		return "gemma4:26b"
	default:
		return ""
	}
}

func gpuBoardBin() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "cloud/sync/code/bin/gpu-board"),
		"gpu-board",
	}
	for _, c := range candidates {
		if filepath.IsAbs(c) {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

func fetchGpuBoardCmd(tag string) tea.Cmd {
	return func() tea.Msg {
		bin := gpuBoardBin()
		if bin == "" {
			return gpuBoardFetchedMsg{tag: tag, line: "GPU: gpu-board not on PATH", failed: true}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "--for", tag, "--line")
		out, err := cmd.Output()
		line := strings.TrimSpace(string(out))
		if err != nil {
			if line == "" {
				line = "GPU: board unavailable (" + err.Error() + ")"
			}
			return gpuBoardFetchedMsg{tag: tag, line: line, failed: true}
		}
		if line == "" {
			line = "GPU: (empty)"
		}
		return gpuBoardFetchedMsg{tag: tag, line: line}
	}
}

func (h *Home) maybeFetchGpuBoard(tool string) tea.Cmd {
	tag := gpuTagForTool(tool)
	if tag == "" {
		return nil
	}
	h.gpuBoardMu.Lock()
	defer h.gpuBoardMu.Unlock()
	if h.gpuBoardFetching == tag {
		return nil
	}
	if t, ok := h.gpuBoardTime[tag]; ok && time.Since(t) < gpuBoardTTL {
		return nil
	}
	h.gpuBoardFetching = tag
	return fetchGpuBoardCmd(tag)
}

func (h *Home) gpuBoardLine(tool string) string {
	tag := gpuTagForTool(tool)
	if tag == "" {
		return ""
	}
	h.gpuBoardMu.RLock()
	line := h.gpuBoardCache[tag]
	h.gpuBoardMu.RUnlock()
	if line == "" {
		line = "GPU: querying hesiod…"
	}
	return line
}

func (h *Home) applyGpuBoardFetched(msg gpuBoardFetchedMsg) {
	h.gpuBoardMu.Lock()
	h.gpuBoardCache[msg.tag] = msg.line
	h.gpuBoardTime[msg.tag] = time.Now()
	if h.gpuBoardFetching == msg.tag {
		h.gpuBoardFetching = ""
	}
	h.gpuBoardMu.Unlock()
}

func renderGpuBoardLine(line string) string {
	if line == "" {
		return ""
	}
	st := lipgloss.NewStyle().Foreground(ColorText)
	return st.Render("GPU  " + line)
}
