package fsm

import (
	"testing"
)

func TestValidateTransition(t *testing.T) {
	sm := NewStateMachine()

	// 合法转换
	if !sm.ValidateTransition(StateINIT, StatePLAN) {
		t.Error("INIT -> PLAN should be valid")
	}
	if !sm.ValidateTransition(StateEXECUTING, StateVERIFYING) {
		t.Error("EXECUTING -> VERIFYING should be valid")
	}
	if !sm.ValidateTransition(StateFAILRetreat, StateEXECUTING) {
		t.Error("FAIL_RETREAT -> EXECUTING should be valid")
	}

	// 非法转换
	if sm.ValidateTransition(StateINIT, StateEXECUTING) {
		t.Error("INIT -> EXECUTING should be invalid")
	}
	if sm.ValidateTransition(StateARCHIVE, StatePLAN) {
		t.Error("ARCHIVE -> PLAN should be invalid")
	}
	if sm.ValidateTransition(StateERROR, StateINIT) {
		t.Error("ERROR -> INIT should be invalid")
	}
}

func TestTransition(t *testing.T) {
	sm := NewStateMachine()

	// 正常转换
	if err := sm.Transition(StatePLAN, "Decider: plan drafted"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm.Current() != StatePLAN {
		t.Errorf("expected current state PLAN, got %s", sm.Current())
	}

	// 继续转换
	if err := sm.Transition(StateEXECUTING, "Reviewer: plan approved, dispatching to executor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sm.Current() != StateEXECUTING {
		t.Errorf("expected current state EXECUTING, got %s", sm.Current())
	}

	// 非法转换 → 强制 ERROR
	if err := sm.Transition(StateARCHIVE, "illegal jump"); err == nil {
		t.Fatal("expected fatal error on illegal transition")
	}
	if sm.Current() != StateERROR {
		t.Errorf("expected forced ERROR state, got %s", sm.Current())
	}

	// 验证日志
	logs := sm.Logs()
	if len(logs) != 3 {
		t.Errorf("expected 3 transition logs, got %d", len(logs))
	}
}

func TestIsTerminal(t *testing.T) {
	sm := NewStateMachine()
	if sm.IsTerminal() {
		t.Error("INIT should not be terminal")
	}

	sm.Transition(StatePLAN, "")
	sm.Transition(StateEXECUTING, "")
	sm.Transition(StateVERIFYING, "")
	sm.Transition(StatePASS, "")
	sm.Transition(StateARCHIVE, "")
	if !sm.IsTerminal() {
		t.Error("ARCHIVE should be terminal")
	}
}

func TestValidTransitionsImmutability(t *testing.T) {
	vt := ValidTransitions()
	// 尝试修改返回的副本
	vt[StateINIT] = []State{StateARCHIVE}
	// 再次获取，原始表不应被修改
	original := ValidTransitions()
	if len(original[StateINIT]) != 1 || original[StateINIT][0] != StatePLAN {
		t.Error("validTransitions should be immutable; mutation leaked")
	}
}
