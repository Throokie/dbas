package demo

import (
	"context"
	"testing"
	"os"
	"time"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/dbas/internal/dbas/persistence"
	"reasonix/internal/provider"
)

func TestMockProvider(t *testing.T) {
	p := mockProvider("test", []string{"response-1", "response-2"})
	if p.Name() != "test" {
		t.Fatalf("Name() = %q, want test", p.Name())
	}
	ch, err := p.Stream(context.Background(), provider.Request{})
	if err != nil { t.Fatal(err) }
	chunk := <-ch
	if chunk.Text != "response-1" {
		t.Errorf("first chunk text = %q, want response-1", chunk.Text)
	}
}

func TestMockRunnerRun(t *testing.T) {
	r := &mockRunner{workdir: t.TempDir()}
	if err := r.Run(context.Background(), "test task"); err != nil { t.Fatal(err) }
	if r.callCount != 1 { t.Errorf("callCount = %d, want 1", r.callCount) }
}

func TestMockRunnerFailure(t *testing.T) {
	r := &mockRunner{failCount: 2, workdir: t.TempDir()}
	if err := r.Run(context.Background(), "task"); err == nil { t.Error("expected error on call 1") }
	if err := r.Run(context.Background(), "task"); err == nil { t.Error("expected error on call 2") }
	if err := r.Run(context.Background(), "task"); err != nil { t.Errorf("expected success, got: %v", err) }
}

func TestBranchLifecycle(t *testing.T) {
	slot := experienceslot.DefaultSlot()
	b := branch.NewBranch("test-branch", nil, "root-hash", "branch-hash", slot)

	if b.CurrentState != fsm.StateINIT {
		t.Errorf("initial state = %v, want INIT", b.CurrentState)
	}
	b.Transition(fsm.StatePLAN, "plan")
	b.Transition(fsm.StateEXECUTING, "exec")
	b.Transition(fsm.StateVERIFYING, "verify")
	b.Transition(fsm.StatePASS, "pass")
	b.Transition(fsm.StateARCHIVE, "archive")
	if !b.IsTerminal() { t.Error("ARCHIVE should be terminal") }
}

func TestBranchIllegalTransition(t *testing.T) {
	slot := experienceslot.DefaultSlot()
	b := branch.NewBranch("test-illegal", nil, "root-hash", "branch-hash", slot)
	err := b.Transition(fsm.StateEXECUTING, "skip plan")
	if err == nil { t.Fatal("expected error for illegal transition INIT -> EXECUTING") }
}

func TestDemoBanner(t *testing.T) {
	if len(banner) == 0 { t.Error("banner should not be empty") }
}

func TestDemoSmoke(t *testing.T) {
	slot := experienceslot.DefaultSlot()
	engine := branch.NewEngine("test-root", slot)
	dir, _ := os.MkdirTemp("", "dbas-demo-test")
	defer os.RemoveAll(dir)
	store := persistence.NewBranchStore(dir)

	RunDemo(engine, store, 19999)
	time.Sleep(200 * time.Millisecond)
}
