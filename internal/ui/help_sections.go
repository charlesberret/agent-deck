package ui

import (
	"strings"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// helpItem is one row of the help overlay: the key label as displayed, what it
// does, and -- when the label names a key the main list can actually receive --
// the trigger to replay so the row can be run from here instead of memorized.
type helpItem struct {
	key     string
	desc    string
	trigger string
}

// runnable reports whether Enter on this row does anything.
func (i helpItem) runnable() bool { return i.trigger != "" }

// helpSection is a titled group of rows.
type helpSection struct {
	title string
	items []helpItem
}

// buildSections assembles the full command list. It was inline in View(); the
// palette needs the same rows for filtering and execution, so it lives here.
func (h *HelpOverlay) buildSections() []helpSection {
	// Define help sections
	newKeys := h.keyPair(hotkeyNewSession, hotkeyQuickCreate, "n/N")
	forkKeys := h.keyPair(hotkeyQuickFork, hotkeyForkWithOptions, "f/F")
	reorderUpKeys := "+ / K / Shift+↑"
	reorderDownKeys := "- / J / Shift+↓"
	indentKeys := "Shift+→/←"
	pinKeys := ","
	searchKey := h.key(hotkeySearch, "/")
	settingsKey := h.key(hotkeySettings, "S")
	helpKey := h.key(hotkeyHelp, "?")
	quitKey := h.key(hotkeyQuit, "q")
	importKey := h.key(hotkeyImport, "i")
	reloadKey := h.key(hotkeyReload, "Ctrl+R")
	deleteKey := h.key(hotkeyDelete, "d")
	closeKey := h.key(hotkeyCloseSession, "D")
	restartKey := h.key(hotkeyRestart, "Shift+R")
	restartFreshKey := h.key(hotkeyRestartFresh, "Shift+T")
	renameKey := h.key(hotkeyRename, "r")
	moveKey := h.key(hotkeyMoveToGroup, "M")
	mcpKey := h.key(hotkeyMCPManager, "m")
	pluginKey := h.key(hotkeyPluginManager, "L")
	skillsKey := h.key(hotkeySkillsManager, "s")
	previewKey := h.key(hotkeyTogglePreview, "v")
	groupViewKey := h.key(hotkeyCycleGroupView, "t")
	timeFilterKey := h.key(hotkeyCycleTimeFilter, "*")
	// Opt-in: empty when switch_session is unbound, so the filter drops the row.
	switchKey := h.key(hotkeySwitchSession, "")
	// In-attach scrollback pager (#1491). Its trigger is resolved directly (it is
	// not a home-screen key); empty label when disabled drops the row.
	scrollbackKey := ResolvedScrollbackTrigger(session.GetHotkeyOverrides()).Label()
	unreadKey := h.key(hotkeyMarkUnread, "u")
	quickApproveKey := h.key(hotkeyQuickApprove, "a")
	promptSessionKey := h.key(hotkeyPromptSession, "o")
	copyKey := h.key(hotkeyCopyOutput, "c")
	copyPaneKey := h.key(hotkeyCopyPane, "V")
	sendKey := h.key(hotkeySendOutput, "x")
	execShellKey := h.key(hotkeyExecShell, "E")
	openShellHereKey := h.key(hotkeyOpenShellHere, "h")
	notesKey := h.key(hotkeyEditNotes, "e")
	if cfg, _ := session.LoadUserConfig(); cfg != nil && !cfg.GetShowNotes() {
		notesKey = ""
	}
	editPathsKey := h.key(hotkeyEditPaths, "p")
	editSessionKey := h.key(hotkeyEditSession, "P")
	worktreeSetupKey := h.key(hotkeyWorktreeSetup, "b")
	worktreeKey := h.key(hotkeyWorktreeFinish, "W")
	watcherPanelKey := h.key(hotkeyWatcherPanel, "w")
	agentsPanelKey := h.key(hotkeyAgentsPanel, "alt+a")
	groupKey := h.key(hotkeyCreateGroup, "g")
	undoKey := h.key(hotkeyUndoDelete, "Ctrl+Z")
	archiveKey := h.key(hotkeyArchiveSession, "A")
	unarchiveKey := h.key(hotkeyUnarchiveSession, "Shift+U")
	viewArchivedKey := h.key(hotkeyViewArchived, "^")
	detachKey := DetachByteLabel(DetachByteFromBinding(h.key(hotkeyDetach, "ctrl+q")))

	sections := []struct {
		title string
		items [][2]string // [key, description]
	}{
		{
			title: "QUICK START",
			items: [][2]string{
				{"Enter", "Attach to selected session"},
				{restartKey, "Restart selected session"},
				{detachKey, "Detach from session"},
				{helpKey, "Open this help"},
			},
		},
		{
			title: "NAVIGATION",
			items: [][2]string{
				{"j / Down", "Move down"},
				{"k / Up", "Move up"},
				{"Ctrl+u/d", "Half page up/down"},
				{"PgUp / PgDn", "Half page up/down"},
				{"Ctrl+f/b", "Full page up/down"},
				{"Home / End", "Jump to first / last item"},
				{"gg / G", "Jump to top / global search"},
				{"h / Left", "Collapse / parent"},
				{"l / Right", "Expand / toggle"},
				{"Shift+Left", "Fully collapse top-level folder (current)"},
				{"Shift+Right", "Fully expand top-level folder (current)"},
				{"Cmd+Shift+Left", "Collapse entire tree"},
				{"Cmd+Shift+Right", "Expand entire tree"},
				{"1-9", "Jump to root group"},
				{"Space", "Jump mode"},
				{"Enter", "Attach / toggle"},
				{"Shift+Enter", "Open session in new iTerm window (macOS)"},
			},
		},
		{
			title: "GROUP NAVIGATION (v1.7.60)",
			items: [][2]string{
				{"Alt+j / Alt+k", "Next / prev session in group"},
				{"Alt+1 - Alt+9", "Jump to Nth session in group"},
				{"Alt+g / Alt+G", "First / last in group"},
				{"Alt+/", "Filter search in group"},
			},
		},
		{
			title: "SESSIONS",
			items: [][2]string{
				{newKeys, "New / quick create"},
				{renameKey, "Rename session"},
				{restartKey, "Restart session"},
				{restartFreshKey, "Restart with new session ID"},
				{deleteKey, "Delete session"},
				{closeKey, "Close session process"},
				{undoKey, "Undo delete"},
				{archiveKey, "Archive session"},
				{unarchiveKey, "Unarchive session"},
				{viewArchivedKey, "Toggle archived view"},
				{moveKey, "Move to group"},
				{mcpKey, "MCP Manager (Claude/Gemini/Cursor)"},
				{pluginKey, "Plugin Manager (Claude — RFC PLUGIN_ATTACH.md)"},
				{skillsKey, "Skills Manager"},
				{CostDashboardKey, "Cost Dashboard"},
				{previewKey, "Toggle preview mode (output/stats/both)"},
				{"O", "Toggle preview orientation (right / below — portrait monitors)"},
				{"< / >", "Shrink / grow preview pane by 5% (drag divider with mouse; vertical in below-orientation)"},
				{unreadKey, "Mark unread"},
				{quickApproveKey, "Quick approve (send '1' to Claude)"},
				{promptSessionKey, "Prompt session (send a one-line prompt without attaching)"},
				{reorderUpKeys, "Reorder up (auto-promote at edge)"},
				{reorderDownKeys, "Reorder down (auto-promote at edge)"},
				{indentKeys, "Indent / outdent (in group)"},
				{pinKeys, "Pin (cycle off→top→bottom→off)"},
				{forkKeys, "Fork session (Claude/Pi)"},
				{copyKey, "Copy output to clipboard"},
				{"C", "Copy preview info (Repo / Path / Branch)"},
				{"Y", "Copy a code block from output"},
				{copyPaneKey, "Copy visible terminal text, including links"},
				{sendKey, "Send output to session"},
				{execShellKey, "Exec shell in sandbox container"},
				{openShellHereKey, "Open shell in session's worktree (split pane / tmux)"},
				{editPathsKey, "Edit multi-repo paths"},
				{editSessionKey, "Edit session settings (title/color/...)"},
				{notesKey, "Edit notes"},
			},
		},
		{
			title: "WORKTREES",
			items: [][2]string{
				{worktreeSetupKey, "Re-run worktree setup script"},
				{worktreeKey, "Finish worktree (merge + cleanup)"},
				{"n → w", "Create session in worktree"},
				{"F → w", "Fork session into worktree"},
			},
		},
		{
			title: "WATCHERS",
			items: [][2]string{
				{watcherPanelKey, "Watcher panel"},
			},
		},
		{
			title: "GROUPS",
			items: [][2]string{
				{groupKey, "New group"},
				{renameKey, "Rename group"},
				{"Tab", "Toggle expand"},
			},
		},
		{
			title: "SEARCH & FILTER",
			items: [][2]string{
				{searchKey, "Open search"},
				{FilterKeyError, "Filter errors"},
				{FilterKeyActive, "Filter open (hide errors)"},
				{"/waiting", "Filter waiting"},
				{"/running", "Filter running"},
				{"/idle", "Filter idle"},
				{groupViewKey, "Cycle view: active-on-top / populated-on-top"},
				{timeFilterKey, "Cycle time filter: today / 3 days / 7 days / all"},
			},
		},
		{
			title: "OTHER",
			items: [][2]string{
				{settingsKey, "Settings"},
				{reloadKey, "Reload from disk"},
				{importKey, "Import tmux sessions"},
				{switchKey, "Switch session (here or attached)"},
				{scrollbackKey, "Scrollback pager (while attached)"},
				{quitKey, "Quit"},
				{helpKey, "This help"},
			},
		},
		{
			title: "STARTUP FLAGS",
			items: [][2]string{
				{"--group <name>", "Launch scoped to a group"},
				{"--profile <name>", "Use specific profile"},
			},
		},
	}

	// The Agents row appears only once something has been adopted. The whole
	// feature is opt-in by presence, and the help overlay is part of the TUI:
	// a user with no agents must not find a key here for a surface that does
	// not exist for them.
	if h.hasAgents {
		for i := range sections {
			if sections[i].title == "WATCHERS" {
				sections[i].items = append(sections[i].items, [2]string{agentsPanelKey, "Agents"})
				break
			}
		}
	}

	for i := range sections {
		filtered := sections[i].items[:0]
		for _, item := range sections[i].items {
			if strings.TrimSpace(item[0]) == "" {
				continue
			}
			filtered = append(filtered, item)
		}
		sections[i].items = filtered
	}

	out := make([]helpSection, 0, len(sections))
	for _, sec := range sections {
		converted := helpSection{title: sec.title}
		for _, item := range sec.items {
			converted.items = append(converted.items, helpItem{
				key:     item[0],
				desc:    item[1],
				trigger: deriveTrigger(item[0]),
			})
		}
		out = append(out, converted)
	}
	return out
}
