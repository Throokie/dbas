package brain

import (
	"context"
	"fmt"
	"os"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/dbas/internal/dbas/persistence"
	"reasonix/dbas/internal/dbas/verifier"
	"reasonix/internal/provider"
)

// mockProvider is a test provider that returns canned responses based on prompt content.
type mockProvider struct {
	name       string
	responses  []string
	callIndex  int
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	go func() {
		defer close(ch)
		if m.callIndex < len(m.responses) {
			ch <- provider.Chunk{Type: provider.ChunkText, Text: m.responses[m.callIndex]}
			m.callIndex++
		}
	}()
	return ch, nil
}

// TestCoordinatorLifecycle runs a complete branch lifecycle with mock providers.
func TestCoordinatorLifecycle(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { os.RemoveAll(root) })
	t.Cleanup(func() { os.RemoveAll(root) })
	store := persistence.NewBranchStore(root)
	slot := experienceslot.DefaultSlot()

	// Decider proposes EXECUTING
	deciderProv := &mockProvider{
		name: "mock-decider",
		responses: []string{
			`{"task_id":"","target_state":"EXECUTING","execution_plan":"write hello world","tools_needed":[],"expected_output":"hello.go","confidence":0.9}`,
		},
	}

	// Reviewer approves with execution input
	reviewerProv := &mockProvider{
		name: "mock-reviewer",
		responses: []string{
			`{"approved":true,"target_state":"EXECUTING","execution_input":"Write a hello world Go program and save it to workspace/hello.go","review_notes":"looks good","modifications":[]}`,
		},
	}

	decider := NewDecider(deciderProv)
	reviewer := NewReviewer(reviewerProv)

	// Mock executor that succeeds
	mockExec := &mockExecutor{}

	verifierTeam := verifier.NewVerifier()

	b := branch.NewBranch("test-lifecycle", nil, "root-hash", "branch-hash", slot)
	if err := store.InitBranch(b); err != nil {
		t.Fatalf("InitBranch: %v", err)
	}

	coord := NewDeciderCoordinator(decider, reviewer, mockExec, verifierTeam, nil, store, b)

	ctx := context.Background()
	if err := coord.Run(ctx, "Write a hello world program"); err != nil {
		t.Fatalf("coordinator Run: %v", err)
	}

	// Verify final state
	if b.CurrentState != fsm.StateARCHIVE {
		t.Fatalf("expected final state ARCHIVE, got %s", b.CurrentState)
	}

	// Verify branch was persisted
	loaded, err := store.LoadBranchMeta(b.BranchID)
	if err != nil {
		t.Fatalf("LoadBranchMeta: %v", err)
	}
	if loaded.CurrentState != fsm.StateARCHIVE {
		t.Fatalf("persisted state = %s, want ARCHIVE", loaded.CurrentState)
	}
}

// TestCoordinatorRetryAndEscalation tests the bug_loop retry logic.
func TestCoordinatorRetryAndEscalation(t *testing.T) {
	root := t.TempDir()
	store := persistence.NewBranchStore(root)
	slot := experienceslot.DefaultSlot()

	deciderProv := &mockProvider{
		name: "mock-decider",
		responses: []string{
			`{"target_state":"EXECUTING","execution_plan":"plan","tools_needed":[],"expected_output":"out","confidence":0.9}`,
		},
	}

	reviewerProv := &mockProvider{
		name: "mock-reviewer",
		responses: []string{
			`{"approved":true,"target_state":"EXECUTING","execution_input":"do it","review_notes":"ok","modifications":[]}`,
		},
	}

	decider := NewDecider(deciderProv)
	reviewer := NewReviewer(reviewerProv)

	// Executor fails twice, then succeeds
	mockExec := &mockExecutor{failCount: 2}

	verifierTeam := verifier.NewVerifier()

	b := branch.NewBranch("test-retry", nil, "root", "branch", slot)
	if err := store.InitBranch(b); err != nil {
		t.Fatalf("InitBranch: %v", err)
	}

	coord := NewDeciderCoordinator(decider, reviewer, mockExec, verifierTeam, nil, store, b)

	ctx := context.Background()
	if err := coord.Run(ctx, "task"); err != nil {
		t.Fatalf("coordinator Run: %v", err)
	}

	if b.CurrentState != fsm.StateARCHIVE {
		t.Fatalf("expected ARCHIVE, got %s", b.CurrentState)
	}
	if mockExec.callCount < 3 {
		t.Fatalf("expected at least 3 executor calls, got %d", mockExec.callCount)
	}
	if b.Retries != 0 {
		t.Fatalf("expected retries reset to 0 after success, got %d", b.Retries)
	}
}

type mockExecutor struct {
	failCount  int
	callCount  int
}

func (m *mockExecutor) Run(ctx context.Context, input string) error {
	m.callCount++
	if m.callCount <= m.failCount {
		return fmt.Errorf("mock executor failure %d", m.callCount)
	}
	return nil
}
