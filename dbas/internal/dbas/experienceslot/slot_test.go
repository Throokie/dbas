package experienceslot

import (
	"testing"

	"reasonix/dbas/internal/dbas/fsm"
)

func TestDefaultSlot(t *testing.T) {
	slot := DefaultSlot()

	if slot.ProjectID != "default-dbas-project" {
		t.Errorf("expected default project ID, got %s", slot.ProjectID)
	}

	if len(slot.DefaultFlow) == 0 {
		t.Fatal("expected at least one default flow state")
	}

	first := slot.DefaultFlow[0]
	if first != fsm.StateINIT {
		t.Errorf("expected first flow state INIT, got %s", first)
	}

	last := slot.DefaultFlow[len(slot.DefaultFlow)-1]
	if last != fsm.StateARCHIVE {
		t.Errorf("expected last flow state ARCHIVE, got %s", last)
	}
}

func TestEscalationThreshold(t *testing.T) {
	slot := DefaultSlot()

	if slot.Rules.EscalationThreshold <= 0 {
		t.Errorf("expected positive escalation threshold, got %d", slot.Rules.EscalationThreshold)
	}

	// Should not escalate below threshold
	if slot.ShouldEscalate(0) {
		t.Error("retries=0 should not trigger escalation")
	}
	if slot.ShouldEscalate(slot.Rules.EscalationThreshold - 1) {
		t.Errorf("retries=%d should not trigger escalation (threshold=%d)",
			slot.Rules.EscalationThreshold-1, slot.Rules.EscalationThreshold)
	}

	// Should escalate at or above threshold
	if !slot.ShouldEscalate(slot.Rules.EscalationThreshold) {
		t.Errorf("retries=%d should trigger escalation", slot.Rules.EscalationThreshold)
	}
	if !slot.ShouldEscalate(slot.Rules.EscalationThreshold + 5) {
		t.Error("high retries should trigger escalation")
	}
}

func TestForceVerifierStates(t *testing.T) {
	slot := DefaultSlot()

	if len(slot.Rules.ForceVerifierForStates) == 0 {
		t.Fatal("expected at least one forced verifier state")
	}

	hasExecuting := false
	for _, s := range slot.Rules.ForceVerifierForStates {
		if s == fsm.StateEXECUTING {
			hasExecuting = true
			break
		}
	}
	if !hasExecuting {
		t.Error("expected EXECUTING to be in force verifier states")
	}
}

func TestVerifierRules(t *testing.T) {
	slot := DefaultSlot()

	if slot.Rules.VerifierRules.CodeSyntax == "" {
		t.Error("expected non-empty code syntax")
	}
	if slot.Rules.VerifierRules.TestPassRate <= 0 || slot.Rules.VerifierRules.TestPassRate > 100 {
		t.Errorf("expected test pass rate between 1-100, got %d", slot.Rules.VerifierRules.TestPassRate)
	}
}

func TestWorkflowRules(t *testing.T) {
	slot := DefaultSlot()

	bugLoop, ok := slot.Workflows["bug_loop"]
	if !ok {
		t.Fatal("expected bug_loop workflow")
	}

	if bugLoop.From != fsm.StateVERIFYING {
		t.Errorf("expected bug_loop.From=VERIFYING, got %s", bugLoop.From)
	}
	if bugLoop.OnFail != fsm.StateEXECUTING {
		t.Errorf("expected bug_loop.OnFail=EXECUTING, got %s", bugLoop.OnFail)
	}
	if bugLoop.MaxRetries <= 0 {
		t.Errorf("expected positive MaxRetries, got %d", bugLoop.MaxRetries)
	}
}

func TestMaxRetriesForWorkflow(t *testing.T) {
	slot := DefaultSlot()

	retries := slot.MaxRetriesForWorkflow("bug_loop")
	if retries <= 0 {
		t.Errorf("expected positive retries for bug_loop, got %d", retries)
	}

	// Non-existent workflow returns 0
	if slot.MaxRetriesForWorkflow("nonexistent") != 0 {
		t.Error("expected 0 retries for non-existent workflow")
	}
}

func TestValidateDecision(t *testing.T) {
	slot := DefaultSlot()

	// Valid: low retries, EXECUTING as target
	slot.Rules.EscalationThreshold = 3
	err := slot.ValidateDecision(fsm.StatePLAN, fsm.StateEXECUTING, 0)
	if err != nil {
		t.Errorf("expected no error for valid decision, got %v", err)
	}

	// Exceeding max retries for bug_loop should error
	slot.Workflows["bug_loop"] = WorkflowRule{
		From:       fsm.StateVERIFYING,
		OnFail:     fsm.StateEXECUTING,
		MaxRetries: 3,
	}
	err = slot.ValidateDecision(fsm.StateVERIFYING, fsm.StateEXECUTING, 10)
	if err == nil {
		t.Error("expected error when retries exceed max")
	}
}

func TestValidateDecisionNoErrorForLowRetries(t *testing.T) {
	slot := DefaultSlot()

	err := slot.ValidateDecision(fsm.StateEXECUTING, fsm.StateVERIFYING, 0)
	if err != nil {
		t.Errorf("expected no error for valid transition with low retries, got %v", err)
	}

	err = slot.ValidateDecision(fsm.StatePLAN, fsm.StateEXECUTING, 2)
	if err != nil {
		t.Errorf("expected no error for retries below max, got %v", err)
	}
}

func TestValidateDecisionWithCustomRule(t *testing.T) {
	slot := DefaultSlot()
	slot.Workflows["custom_flow"] = WorkflowRule{
		From:       fsm.StateFAILRetreat,
		OnFail:     fsm.StateERROR,
		MaxRetries: 2,
	}

	// Should error if retries exceed max for this workflow
	err := slot.ValidateDecision(fsm.StateFAILRetreat, fsm.StateERROR, 5)
	if err == nil {
		t.Error("expected error for retries exceeding custom workflow max")
	}

	// Should not error for a different transition
	err = slot.ValidateDecision(fsm.StateEXECUTING, fsm.StateVERIFYING, 5)
	if err != nil {
		t.Errorf("expected no error for different transition, got %v", err)
	}
}

func TestToJSONAndFromJSON(t *testing.T) {
	slot := DefaultSlot()

	data, err := slot.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	parsed, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if parsed.ProjectID != slot.ProjectID {
		t.Errorf("project ID round-trip: expected %s, got %s", slot.ProjectID, parsed.ProjectID)
	}
	if parsed.Rules.EscalationThreshold != slot.Rules.EscalationThreshold {
		t.Errorf("escalation threshold round-trip: expected %d, got %d",
			slot.Rules.EscalationThreshold, parsed.Rules.EscalationThreshold)
	}

	// Verify workflow round-trip
	for name, rule := range slot.Workflows {
		parsedRule, ok := parsed.Workflows[name]
		if !ok {
			t.Errorf("workflow %s not found after round-trip", name)
			continue
		}
		if parsedRule.MaxRetries != rule.MaxRetries {
			t.Errorf("workflow %s MaxRetries: expected %d, got %d", name, rule.MaxRetries, parsedRule.MaxRetries)
		}
	}
}

func TestFromJSONInvalidData(t *testing.T) {
	_, err := FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestForceVerifierForStatesEmpty(t *testing.T) {
	slot := DefaultSlot()
	slot.Rules.ForceVerifierForStates = []fsm.State{}

	// This should still validate — it just won't force verifier for any state
	err := slot.ValidateDecision(fsm.StateEXECUTING, fsm.StateVERIFYING, 0)
	if err != nil {
		t.Errorf("expected no error with empty force verifier states, got %v", err)
	}
}

func TestDefaultFlowIntegrity(t *testing.T) {
	slot := DefaultSlot()

	// Default flow should be: INIT -> PLAN -> EXECUTING -> VERIFYING -> ARCHIVE
	expectedFlow := []fsm.State{
		fsm.StateINIT, fsm.StatePLAN, fsm.StateEXECUTING,
		fsm.StateVERIFYING, fsm.StateARCHIVE,
	}

	if len(slot.DefaultFlow) != len(expectedFlow) {
		t.Fatalf("expected %d flow states, got %d", len(expectedFlow), len(slot.DefaultFlow))
	}

	for i, expected := range expectedFlow {
		if slot.DefaultFlow[i] != expected {
			t.Errorf("flow state %d: expected %s, got %s", i, expected, slot.DefaultFlow[i])
		}
	}
}
