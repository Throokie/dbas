package branch

import (
	"testing"

	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
)

func TestEngineCreateAndGet(t *testing.T) {
	slot := experienceslot.DefaultSlot()
	eng := NewEngine("root-hash-123", slot)

	b, err := eng.CreateBranch("api-gateway-refactor", nil)
	if err != nil {
		t.Fatalf("create branch failed: %v", err)
	}
	if b == nil {
		t.Fatal("branch is nil")
	}
	if b.ParentBranchID != nil {
		t.Error("root branch should have nil parent")
	}

	// 查询
	found, ok := eng.GetBranch(b.BranchID)
	if !ok {
		t.Fatal("branch not found after creation")
	}
	if found.BranchID != b.BranchID {
		t.Error("branch ID mismatch")
	}

	// 状态检查
	if found.CurrentState != fsm.StateINIT {
		t.Errorf("expected INIT, got %s", found.CurrentState)
	}
	if found.ExecutorInfo.Model != "default-agent" {
		t.Errorf("expected default-agent, got %s", found.ExecutorInfo.Model)
	}
}

func TestEngineChildBranch(t *testing.T) {
	eng := NewEngine("root-hash", experienceslot.DefaultSlot())

	parent, _ := eng.CreateBranch("parent-task", nil)
	parentID := parent.BranchID

	child, err := eng.CreateBranch("child-task", &parentID)
	if err != nil {
		t.Fatalf("create child branch failed: %v", err)
	}
	if child.ParentBranchID == nil || *child.ParentBranchID != parentID {
		t.Error("child branch parent ID mismatch")
	}
	if child.HashingChain.RootHash != "root-hash" {
		t.Error("child should inherit root hash")
	}
}

func TestEngineResetBranch(t *testing.T) {
	eng := NewEngine("root-hash", experienceslot.DefaultSlot())
	b, _ := eng.CreateBranch("test", nil)

	// 模拟执行到 EXECUTING
	b.Transition(fsm.StatePLAN, "plan")
	b.Transition(fsm.StateEXECUTING, "execute")
	b.IncrementRetries()
	b.IncrementRetries()

	// 重置
	if err := eng.ResetBranch(b.BranchID); err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	if b.CurrentState != fsm.StateINIT {
		t.Errorf("expected INIT after reset, got %s", b.CurrentState)
	}
	if b.Retries != 0 {
		t.Errorf("expected retries=0, got %d", b.Retries)
	}
	if b.ExecutorInfo.Model != "default-agent" {
		t.Errorf("expected model reset to default-agent, got %s", b.ExecutorInfo.Model)
	}
}

func TestEngineGetActiveBranches(t *testing.T) {
	eng := NewEngine("root-hash", experienceslot.DefaultSlot())
	b1, _ := eng.CreateBranch("active", nil)
	b2, _ := eng.CreateBranch("terminal", nil)

	// b2 进入 ARCHIVE
	b2.Transition(fsm.StatePLAN, "")
	b2.Transition(fsm.StateEXECUTING, "")
	b2.Transition(fsm.StateVERIFYING, "")
	b2.Transition(fsm.StatePASS, "")
	b2.Transition(fsm.StateARCHIVE, "")

	active := eng.GetActiveBranches()
	if len(active) != 1 {
		t.Errorf("expected 1 active branch, got %d", len(active))
	}
	if active[0].BranchID != b1.BranchID {
		t.Error("active branch should be b1")
	}
}
