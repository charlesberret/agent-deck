package ui

import (
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/lipgloss"
)

// connectionStatusLine decides the preview-pane "Status:" line for a tool
// section whose conversation/session id is on record.
//
// A tool's session id is retained even after a session is archived or stopped
// (so the conversation can be resumed), so its mere presence does NOT mean the
// agent is live. Archived and stopped sessions have had their tmux pane torn
// down and must not claim "Connected".
func connectionStatusLine(archived bool, status session.Status) (text string, style lipgloss.Style) {
	switch {
	case archived:
		return "■ Archived", SessionStatusStopped
	case status == session.StatusStopped:
		return "■ Stopped", SessionStatusStopped
	default:
		return "● Connected", lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	}
}

// rowStatusGlyph decides the session-list row status indicator (glyph + style).
//
// The coarse status comes from a render snapshot of the session's last-known
// state. Archiving tears down the tmux pane but does NOT reset Status, so an
// archived row would otherwise keep a live glyph (e.g. ● running); the archived
// override forces the stopped glyph regardless of the stale status/substate.
//
// IconAuthHold is a monochrome lock (not emoji 🔒) for auth-held sessions.
// Matches house monochrome tool icons (▣ ✦ ⌂); renders as text, not color emoji.
const IconAuthHold = "⚿" // U+26BF SQUARED KEY

// rowStatusGlyph decides the session-list row status indicator (glyph + style).
//
// The coarse status comes from a render snapshot of the session's last-known
// state. Archiving tears down the tmux pane but does NOT reset Status, so an
// archived row would otherwise keep a live glyph (e.g. ● running); the archived
// override forces the stopped glyph regardless of the stale status/substate.
//
// neverStarted is true when LastStartedAt is zero (belt-and-braces with
// StatusNeverStarted). Prefer status never_started; fall back to neverStarted
// for legacy error rows not yet bulk-normalized.
func rowStatusGlyph(status session.Status, substate session.Substate, archived bool, neverStarted bool) (icon string, style lipgloss.Style) {
	// Explicit never_started status (post bulk-normalize / UpdateStatus).
	if status == session.StatusNeverStarted && !archived {
		return "·", SessionStatusIdle
	}
	// Legacy: error/stopped with no LastStartedAt → dormant, not alarm.
	// Idle is a live-at-rest status ("was up, now quiet") and must not
	// collapse to the unstarted middle-dot — that is the ●→· drop the
	// house reads as "this seat died."
	if neverStarted && !archived {
		switch status {
		case session.StatusRunning, session.StatusWaiting, session.StatusIdle:
			// Live or at-rest seats keep their real glyph.
		default:
			return "·", SessionStatusIdle
		}
	}

	switch status {
	case session.StatusRunning:
		icon, style = "●", SessionStatusRunning
	case session.StatusWaiting:
		icon, style = "◐", SessionStatusWaiting
	case session.StatusIdle:
		icon, style = "○", SessionStatusIdle
	case session.StatusError:
		icon, style = "✕", SessionStatusError
	case session.StatusStopped:
		icon, style = "■", SessionStatusStopped
	case session.StatusNeverStarted:
		icon, style = "·", SessionStatusIdle
	default:
		icon, style = "○", SessionStatusIdle
	}

	// Honest Status v2: a distinct glyph for the two error substates a
	// supervisor must act on differently — a dead-model no-op loop and an
	// auth/login failure both render as "error", but a generic "✕" hides which.
	// "⚡" = model unavailable (the Fable-down no-op); monochrome ⚿ = auth/login.
	// Gated on StatusError so a stale cached substate cannot leak the glyph onto
	// a session that is no longer in error.
	if status == session.StatusError {
		switch substate {
		case session.SubstateModelUnavailable:
			icon = "⚡"
		case session.SubstateAuth401:
			icon = IconAuthHold
		}
	}

	// A stopped session can ALSO need auth: an agent that exits on a 401 may exit
	// cleanly (exit 0), which the exit-code classifier reads as stopped (■). That
	// is the silent-decay shape of the 2026-07-26 fleet death — sessions quietly
	// going ■ with nothing saying "your token is broken". The auth-401 substate
	// only reaches a stopped row when an auth hold is live (a deliberate stop
	// releases the hold), so this cannot resurrect a stale glyph.
	if status == session.StatusStopped && substate == session.SubstateAuth401 {
		icon = IconAuthHold
	}

	if archived {
		icon, style = "■", SessionStatusStopped
	}
	return icon, style
}

// remoteRowStatusGlyph is the remote-session entry point to rowStatusGlyph.
//
// A remote row's state arrives as raw JSON strings from `agent-deck list --json`
// on the far side (session.RemoteSessionInfo), never as typed values. Those
// strings are emitted from the same session.Status / tmux.Substate constants
// (see StatusString in cmd/agent-deck/cli_utils.go), so the conversion is a
// direct cast; an unrecognized value — including the empty string an older
// remote sends for a field it predates — falls through to rowStatusGlyph's
// ○ idle default.
func remoteRowStatusGlyph(status, substate string, archived bool) (icon string, style lipgloss.Style) {
	return rowStatusGlyph(session.Status(status), session.Substate(substate), archived, false)
}

// statusMotion is how a live row should breathe. Shape stays a filled
// circle for any live seat; only the motion (and brightness) change.
type statusMotion int

const (
	motionNone statusMotion = iota
	motionCrunch            // generating / tool-use — continuous pulse
	motionListen            // live loop at rest — occasional blip
)

const (
	motionTickInterval = 400 * time.Millisecond
	crunchPulseFrames  = 4
	listenBlipEvery    = 10 // one bright frame every ~4s
)

// classifyStatusMotion decides whether a row should pulse, blip, or sit still.
// Archived / never-started / error / stopped / waiting (needs eyes) stay still.
func classifyStatusMotion(status session.Status, substate session.Substate, archived, neverStarted bool) statusMotion {
	if archived {
		return motionNone
	}
	if status == session.StatusNeverStarted {
		return motionNone
	}
	if neverStarted {
		switch status {
		case session.StatusRunning, session.StatusWaiting, session.StatusIdle:
			// fall through — live-or-at-rest even if LastStartedAt is missing
		default:
			return motionNone
		}
	}
	switch status {
	case session.StatusRunning:
		if substate == session.SubstateIdleAtEmptyPrompt {
			return motionListen
		}
		return motionCrunch
	case session.StatusIdle:
		return motionListen
	default:
		return motionNone
	}
}

// applyStatusMotion keeps the glyph family stable (live = filled ●) and
// varies only color. Crunching breathes; a listener stays muted and ticks
// once in a while so it cannot be mistaken for the unstarted ·.
func applyStatusMotion(icon string, style lipgloss.Style, motion statusMotion, frame int) (string, lipgloss.Style) {
	switch motion {
	case motionCrunch:
		step := frame % crunchPulseFrames
		st := lipgloss.NewStyle().Foreground(ColorGreen)
		switch step {
		case 0:
			st = st.Bold(true)
		case 2:
			st = lipgloss.NewStyle().Foreground(ColorComment)
		}
		return "●", st
	case motionListen:
		if listenBlipEvery > 0 && frame%listenBlipEvery == 0 {
			return "●", lipgloss.NewStyle().Foreground(ColorGreen)
		}
		return "●", lipgloss.NewStyle().Foreground(ColorComment)
	default:
		return icon, style
	}
}

// authHoldHintLines is the preview-pane hint for a session held out of automatic
// restarts because its agent cannot authenticate.
//
// Deliberately spells out BOTH halves the 2026-07-26 outage lacked: why nothing
// is retrying (so the silence does not read as agent-deck being broken), and the
// exact two-step remedy. "Press R" is the escape hatch — the TUI restart is
// never gated, because a human pressing it IS the "credentials are fixed" signal.
var authHoldHintLines = []string{
	IconAuthHold + " auth failed — automatic restarts are held",
	"   a restart can't fix a credential, and each one races the shared token",
	"   fix: run /login (or repair credentials), then press R here",
}

// authHoldBannerLines renders the hint, each line truncated to width.
func authHoldBannerLines(width int) string {
	headStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	out := ""
	for i, line := range authHoldHintLines {
		style := bodyStyle
		if i == 0 {
			style = headStyle
		}
		out += style.MaxWidth(max(width, 1)).Render(line) + "\n"
	}
	return out
}
