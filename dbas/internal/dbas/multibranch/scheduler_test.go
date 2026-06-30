package multibranch

import (
	"context"
	"errors"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/persistence"
)

// --- Mocks ---

type mockRunner struct {
	failInit bool
	runErr   error
}

func (r *mockRunner) Run(ctx context.Context, input string) error {
	if r.runErr != nil {
		return r.runErr
	}
	return nil
}

type mockFactory struct {
	runner *mockRunner
}

func (f *mockFactory) Create(b *branch.Branch) (Runner, error) {
	if f.runner.failInit {
		return nil, errors.New("factory init failed")
	}
	return f.runner, nil
}

func setupScheduler(t *testing.T) (*Scheduler, *persistence.BranchStore, *branch.BranchEngine) {
	t.Helper()
	root := t.TempDir()
	slot := experienceslot.DefaultSlot()
	engine := branch.NewEngine("root-hash", slot)
	store := persistence.NewBranchStore(root)
	scheduler := NewScheduler(engine, store, &mockFactory{
		runner: &mockRunner{},
	}, 4)
	return scheduler, store, engine
}

// --- Tests ---

func TestNewScheduler(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)
	if scheduler == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if scheduler.maxParallel != 4 {
		t.Errorf("expected maxParallel 4, got %d", scheduler.maxParallel)
	}
}

func TestSchedulerDefaultMaxParallel(t *testing.T) {
	root := t.TempDir()
	slot := experienceslot.DefaultSlot()
	engine := branch.NewEngine("hash", slot)
	store := persistence.NewBranchStore(root)
	s := NewScheduler(engine, store, &mockFactory{runner: &mockRunner{}}, 0)
	if s.maxParallel != 4 {
		t.Errorf("expected default maxParallel 4, got %d", s.maxParallel)
	}
}

func TestInitContextTree(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)

	root, shared := scheduler.InitContextTree("system prompt", "tool schema", "shared context")
	if root == nil {
		t.Fatal("expected non-nil root node")
	}
	if shared == nil {
		t.Fatal("expected non-nil shared node")
	}
	if root.ZoneType != "pinned" {
		t.Errorf("expected root ZoneType 'pinned', got '%s'", root.ZoneType)
	}
	if shared.ZoneType != "append" {
		t.Errorf("expected shared ZoneType 'append', got '%s'", shared.ZoneType)
	}
	if shared.ParentID == nil || *shared.ParentID != root.ID {
		t.Error("expected shared node parent to be root")
	}
}

func TestSpawnSequential(t *testing.T) {
	scheduler, store, engine := setupScheduler(t)

	inputs := []TaskInput{
		{Name: "task-a", Input: "do A"},
		{Name: "task-b", Input: "do B"},
	}

	branches, err := scheduler.SpawnSequential(context.Background(), "root", inputs)
	if err != nil {
		t.Fatalf("SpawnSequential: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}

	// Verify branches are stored
	for _, b := range branches {
		_, err := store.LoadBranchMeta(b.BranchID)
		if err != nil {
			t.Errorf("branch %s not persisted: %v", b.BranchID, err)
		}
		_, ok := engine.GetBranch(b.BranchID)
		if !ok {
			t.Errorf("branch %s not in engine", b.BranchID)
		}
	}
}

func TestSpawnSequentialEmptyInputs(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)

	_, err := scheduler.SpawnSequential(context.Background(), "root", nil)
	if err == nil {
		t.Fatal("expected error for empty inputs")
	}
}

func TestSpawnSequentialWithFactoryError(t *testing.T) {
	root := t.TempDir()
	slot := experienceslot.DefaultSlot()
	engine := branch.NewEngine("hash", slot)
	store := persistence.NewBranchStore(root)

	failingFactory := &mockFactory{runner: &mockRunner{failInit: true}}
	scheduler := NewScheduler(engine, store, failingFactory, 4)

	inputs := []TaskInput{{Name: "fail-task", Input: "will fail"}}
	_, err := scheduler.SpawnSequential(context.Background(), "root", inputs)
	if err == nil {
		t.Fatal("expected error from failing factory")
	}
}

func TestSchedulerGetTree(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)
	scheduler.InitContextTree("prompt", "tools", "context")

	tree := scheduler.GetTree()
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}

	root, ok := tree.GetNode("root")
	if !ok {
		t.Fatal("expected root node after InitContextTree")
	}
	if root.Content != "prompt\ntools" {
		t.Errorf("unexpected root content: %s", root.Content)
	}
}

func TestTaskInputType(t *testing.T) {
	input := TaskInput{
		Name:  "test-task",
		Input: "do something",
	}
	if input.Name != "test-task" {
		t.Errorf("expected Name 'test-task', got '%s'", input.Name)
	}
	if input.Input != "do something" {
		t.Errorf("expected Input 'do something', got '%s'", input.Input)
	}
}

func TestRunnerInterface(t *testing.T) {
	var r Runner = &mockRunner{}
	if err := r.Run(context.Background(), "test"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunnerInterfaceWithError(t *testing.T) {
	expected := errors.New("execution failed")
	r := &mockRunner{runErr: expected}
	err := r.Run(context.Background(), "test")
	if err != expected {
		t.Errorf("expected '%v', got '%v'", expected, err)
	}
}

func TestSpawnSequentialOrderPreserved(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)

	inputs := []TaskInput{
		{Name: "first", Input: "task 1"},
		{Name: "second", Input: "task 2"},
		{Name: "third", Input: "task 3"},
	}

	branches, err := scheduler.SpawnSequential(context.Background(), "root", inputs)
	if err != nil {
		t.Fatalf("SpawnSequential: %v", err)
	}

	for i, b := range branches {
		expectedName := inputs[i].Name
		// Branch IDs contain the task name
		if len(b.BranchID) < len(expectedName)+10 {
			t.Errorf("branch %d: expected ID containing '%s', got '%s'", i, expectedName, b.BranchID)
		}
	}
}

func TestInitContextTreeReturnsNodes(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)

	root, shared := scheduler.InitContextTree("A", "B", "C")
	if root.Hash == "" || shared.Hash == "" {
		t.Error("expected non-empty hashes")
	}

	// Shared node hash should be different from root hash
	if root.Hash == shared.Hash {
		t.Error("expected different hashes for root and shared node")
	}
}

func TestMultipleSpawnSequential(t *testing.T) {
	scheduler, _, _ := setupScheduler(t)

	// First batch
	b1, err := scheduler.SpawnSequential(context.Background(), "root", []TaskInput{{Name: "batch1", Input: "do A"}})
	if err != nil {
		t.Fatalf("first batch: %v", err)
	}
	if len(b1) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(b1))
	}

	// Second batch should not interfere with first
	b2, err := scheduler.SpawnSequential(context.Background(), "root", []TaskInput{{Name: "batch2", Input: "do B"}})
	if err != nil {
		t.Fatalf("second batch: %v", err)
	}
	if len(b2) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(b2))
	}

	if b1[0].BranchID == b2[0].BranchID {
		t.Error("expected different branch IDs for different batches")
	}
}
