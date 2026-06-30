package kanban

import (
	"testing"
	"time"

	"reasonix/dbas/internal/dbas/experienceslot"
)

type mockNotifier struct{}

func (m *mockNotifier) Notify(card *KanbanCard) error { return nil }

func TestConsoleNotifier(t *testing.T) {
	n := &ConsoleNotifier{}
	card := &KanbanCard{
		ProjectID: "test-project",
		Timestamp: time.Now().Unix(),
	}
	err := n.Notify(card)
	if err != nil {
		t.Fatalf("ConsoleNotifier.Notify failed: %v", err)
	}
}

func TestKanbanCardBasic(t *testing.T) {
	card := &KanbanCard{
		ProjectID:     "test-proj",
		Timestamp:     1234567890,
		TerminalCount: 5,
	}
	if card.ProjectID != "test-proj" {
		t.Errorf("expected ProjectID 'test-proj', got %q", card.ProjectID)
	}
	if card.TerminalCount != 5 {
		t.Errorf("expected TerminalCount 5, got %d", card.TerminalCount)
	}
}

func TestBranchView(t *testing.T) {
	bv := BranchView{
		ID:    "branch-1",
		Name:  "Test Branch",
		State: "PASS",
		Model: "gpt-4",
	}
	if bv.ID != "branch-1" {
		t.Errorf("expected ID 'branch-1', got %q", bv.ID)
	}
	if bv.State != "PASS" {
		t.Errorf("expected State 'PASS', got %q", bv.State)
	}
}

func TestFeishuCardFormatter(t *testing.T) {
	f := &FeishuCardFormatter{}
	card := &KanbanCard{
		ProjectID: "test",
		Timestamp: time.Now().Unix(),
		Slot:      experienceslot.DefaultSlot(),
	}
	result := f.Format(card)
	if result == "" {
		t.Fatal("expected non-empty formatted card")
	}
}

func TestMockNotifier(t *testing.T) {
	n := &mockNotifier{}
	card := &KanbanCard{ProjectID: "test"}
	err := n.Notify(card)
	if err != nil {
		t.Fatalf("mockNotifier.Notify failed: %v", err)
	}
}
