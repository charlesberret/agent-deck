package ui

import (
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

func TestUnnamedSessionMarker_AutoNameRow(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30
	home.refreshSessionRenderSnapshot(nil)

	inst := session.NewInstanceWithTool("misty-owl", "/tmp/unnamed-marker", "claude")
	inst.SetAutoName(true)
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.refreshSessionRenderSnapshot(home.instances)
	home.rebuildFlatItems()

	var item session.Item
	for _, it := range home.flatItems {
		if it.Type == session.ItemTypeSession && it.Session != nil && it.Session.ID == inst.ID {
			item = it
			break
		}
	}
	if item.Session == nil {
		t.Fatal("session row not in flatItems")
	}

	var b strings.Builder
	home.renderSessionItem(&b, item, false, home.getSessionRenderSnapshot(), 100)
	out := b.String()
	if !strings.Contains(stripANSILatency(out), "~ ") {
		t.Fatalf("auto-named row missing unnamed marker: %q", out)
	}
}

func TestUnnamedSessionMarker_ExplicitTitleHasNoTilde(t *testing.T) {
	home := NewHome()
	home.width = 120
	home.height = 30

	inst := session.NewInstanceWithTool("alchemist", "/tmp/named-marker", "claude")
	inst.SetAutoName(false)
	inst.TitleLocked = true
	home.instances = []*session.Instance{inst}
	home.groupTree = session.NewGroupTree(home.instances)
	home.refreshSessionRenderSnapshot(home.instances)
	home.rebuildFlatItems()

	var item session.Item
	for _, it := range home.flatItems {
		if it.Type == session.ItemTypeSession && it.Session != nil && it.Session.ID == inst.ID {
			item = it
			break
		}
	}
	if item.Session == nil {
		t.Fatal("session row not in flatItems")
	}

	var b strings.Builder
	home.renderSessionItem(&b, item, false, home.getSessionRenderSnapshot(), 100)
	out := stripANSILatency(b.String())
	if strings.Contains(out, "~ ") {
		t.Fatalf("explicitly named row should not show unnamed marker: %q", out)
	}
	if !strings.Contains(out, "alchemist") {
		t.Fatalf("named row missing title: %q", out)
	}
}
