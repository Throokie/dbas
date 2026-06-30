package brain

import (
	"context"
	"fmt"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/executor"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/dbas/internal/dbas/persistence"
	"reasonix/dbas/internal/dbas/verifier"
)


// DeciderCoordinator is the dual-brain coordinator for DBAS, layered on top of Reasonix Coordinator.
// It implements the Run(ctx, input) method, callable directly by CLI or HTTP layers.
type DeciderCoordinator struct {
	decider  *Decider
	reviewer *Reviewer
	executor interface {
		Run(ctx context.Context, input string) error
	}
	verifier    *verifier.Verifier
	llmVerifier *verifier.LLMVerifier
	store       *persistence.BranchStore

	branch *branch.Branch
	slot   *experienceslot.ExperienceSlot
}

func NewDeciderCoordinator(
	decider *Decider,
	reviewer *Reviewer,
	executor interface {
		Run(ctx context.Context, input string) error
	},
	verifier *verifier.Verifier,
	llmVerifier *verifier.LLMVerifier,
	store *persistence.BranchStore,
	b *branch.Branch,
) *DeciderCoordinator {
	return &DeciderCoordinator{
		decider:     decider,
		reviewer:    reviewer,
		executor:    executor,
		verifier:    verifier,
		llmVerifier: llmVerifier,
		store:       store,
		branch:      b,
		slot:        b.Slot(),
	}
}

func (d *DeciderCoordinator) Run(ctx context.Context, input string) error {
	if err := d.store.SaveBranchMeta(d.branch); err != nil {
		return fmt.Errorf("persist branch init: %w", err)
	}
	if err := d.branch.Transition(fsm.StatePLAN, "drafting task plan"); err != nil {
		return fmt.Errorf("phase PLAN: %w", err)
	}
	d.flushLog()

	proposal, err := d.decider.Propose(ctx, input, d.branch)
	if err != nil {
		d.branch.FSM().ForceError(fmt.Sprintf("Decider failed: %v", err))
		return fmt.Errorf("decider: %w", err)
	}
	d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[DECIDER] target=%s, confidence=%.2f", proposal.TargetState, proposal.Confidence))
	d.flushLog()

	decision, err := d.reviewer.Review(ctx, proposal, d.branch, d.slot)
	if err != nil {
		d.branch.FSM().ForceError(fmt.Sprintf("Reviewer failed: %v", err))
		return fmt.Errorf("reviewer: %w", err)
	}
	d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[REVIEWER] Approved=%v, target=%s", decision.Approved, decision.TargetState))
	d.flushLog()

	if err := d.branch.Transition(decision.TargetState, "FSM after reviewer approval"); err != nil {
		return fmt.Errorf("fsm: %w", err)
	}
	d.flushLog()

	if decision.TargetState != fsm.StateEXECUTING {
		if decision.TargetState == fsm.StateARCHIVE {
			return d.archive(ctx, "No execution needed")
		}
		return fmt.Errorf("unexpected target: %s", decision.TargetState)
	}
	return d.executeBugLoop(ctx, decision.ExecutionInput)
}

func (d *DeciderCoordinator) executeBugLoop(ctx context.Context, executionInput string) error {
	maxRetries := d.slot.MaxRetriesForWorkflow("bug_loop")
	if maxRetries <= 0 {
		maxRetries = 3
	}
	for {
		if d.branch.CurrentState != fsm.StateEXECUTING {
			if err := d.branch.Transition(fsm.StateEXECUTING, fmt.Sprintf("attempt %d", d.branch.Retries+1)); err != nil {
				return fmt.Errorf("EXECUTING: %w", err)
			}
			d.flushLog()
		}

		execErr := d.runExecutor(ctx, executionInput)
		if execErr != nil {
			d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[EXECUTOR_ERROR] %v", execErr))
			if err := d.branch.Transition(fsm.StateVERIFYING, "exec error -> VERIFYING"); err != nil {
				return fmt.Errorf("VERIFYING: %w", err)
			}
			d.flushLog()
			if err := d.branch.Transition(fsm.StateFAILRetreat, "exec error -> FAIL_RETREAT"); err != nil {
				return fmt.Errorf("FAIL_RETREAT: %w", err)
			}
			d.flushLog()

			if d.branch.Retries >= maxRetries {
				if d.branch.ExecutorInfo.Model != "default-agent-fallback" {
					d.branch.Escalate()
					d.branch.ResetRetries()
					continue
				}
				if err := d.branch.Transition(fsm.StateERROR, "Max retries exceeded"); err != nil {
					return err
				}
				d.flushLog()
				return fmt.Errorf("max retries exceeded for %s", d.branch.BranchID)
			}
			d.branch.IncrementRetries()
			if d.slot.ShouldEscalate(d.branch.Retries) {
				d.branch.Escalate()
			}
			continue
		}

		if err := d.branch.Transition(fsm.StateVERIFYING, "exec OK -> VERIFYING"); err != nil {
			return fmt.Errorf("VERIFYING: %w", err)
		}
		d.flushLog()

		verdict, err := d.runVerification(ctx)
		if err != nil {
			d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[VERIFIER_ERROR] %v", err))
			continue
		}
		if verdict.Passed {
			d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[VERIFIER_PASS] %s", verdict.Reason))
			if err := d.branch.Transition(fsm.StatePASS, "All checks passed"); err != nil {
				return err
			}
			d.branch.ResetRetries()
			return d.archive(ctx, verdict.Reason)
		}

		if err := d.branch.Transition(fsm.StateFAILRetreat, fmt.Sprintf("verify fail: %s", verdict.Reason)); err != nil {
			return err
		}
		d.flushLog()

		if d.branch.Retries >= maxRetries {
			if d.branch.ExecutorInfo.Model != "qwen-pro-fallback" {
				d.branch.Escalate()
				d.branch.ResetRetries()
				continue
			}
			if err := d.branch.Transition(fsm.StateERROR, "Max retries exceeded"); err != nil {
				return err
			}
			d.flushLog()
			return fmt.Errorf("max retries exceeded for %s", d.branch.BranchID)
		}
		d.branch.IncrementRetries()
		if d.slot.ShouldEscalate(d.branch.Retries) {
			d.branch.Escalate()
		}
	}
}

func (d *DeciderCoordinator) runExecutor(ctx context.Context, executionInput string) error {
	if taskRunner, ok := d.executor.(executor.TaskRunner); ok {
		return taskRunner.RunTask(ctx, executor.NewTask(d.branch, executionInput))
	}
	return d.executor.Run(ctx, executionInput)
}

func (d *DeciderCoordinator) runVerification(ctx context.Context) (*verifier.Verdict, error) {
	verdict, err := d.verifier.Verify(d.branch)
	if err != nil {
		return nil, err
	}
	if !verdict.Passed {
		return verdict, nil
	}
	if d.llmVerifier != nil {
		d.branch.Logs = append(d.branch.Logs, "[LLM_AUDIT] Running cross-audit...")
		d.flushLog()
		llmVerdict, err := d.llmVerifier.CrossAudit(ctx, d.branch)
		if err != nil {
			d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[LLM_AUDIT_ERROR] %v", err))
			d.flushLog()
			return nil, err
		}
		if !llmVerdict.Passed {
			return llmVerdict, nil
		}
		d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[LLM_AUDIT_PASS] %s", llmVerdict.Reason))
		d.flushLog()
	}
	return verdict, nil
}

func (d *DeciderCoordinator) handleExecutionFailure(ctx context.Context, executionInput, reason string) error {
	maxRetries := d.slot.MaxRetriesForWorkflow("bug_loop")
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if d.branch.Retries >= maxRetries {
		if d.branch.ExecutorInfo.Model != "qwen-pro-fallback" {
			d.branch.Escalate()
			d.branch.ResetRetries()
			return nil
		}
		if err := d.branch.Transition(fsm.StateERROR, "Exec failed: "+reason); err != nil {
			return err
		}
		d.flushLog()
		return fmt.Errorf("exec failed: %s", reason)
	}
	d.branch.IncrementRetries()
	if d.slot.ShouldEscalate(d.branch.Retries) {
		d.branch.Escalate()
	}
	return nil
}

func (d *DeciderCoordinator) generateBugFeedback(v *verifier.Verdict) string {
	feedback := "The previous execution failed verification.\n\n"
	feedback += "Failure Reason: " + v.Reason + "\n\n"
	if len(v.Details) > 0 {
		feedback += "Details:\n"
		for _, detail := range v.Details {
			feedback += "- " + detail + "\n"
		}
	}
	feedback += "\nPlease fix and retry."
	return feedback
}

func (d *DeciderCoordinator) archive(ctx context.Context, reason string) error {
	// Ensure we transition through PASS if needed (VERIFYING -> PASS -> ARCHIVE, or directly PLAN/PASS -> ARCHIVE)
	if d.branch.CurrentState == fsm.StateVERIFYING || d.branch.CurrentState == fsm.StatePLAN || d.branch.CurrentState == fsm.StateINIT {
		if err := d.branch.Transition(fsm.StatePASS, "Auto-pass before archive: "+reason); err != nil {
			return err
		}
	}
	if err := d.branch.Transition(fsm.StateARCHIVE, "Archived: "+reason); err != nil {
		return err
	}
	d.branch.Logs = append(d.branch.Logs, fmt.Sprintf("[ARCHIVE] %s", reason))
	d.flushLog()
	return d.store.SaveBranchMeta(d.branch)
}

func (d *DeciderCoordinator) flushLog() {
	if d.store != nil && d.branch != nil && len(d.branch.Logs) > 0 {
		_ = d.store.AppendLog(d.branch.BranchID, d.branch.Logs[len(d.branch.Logs)-1])
	}
}

func (d *DeciderCoordinator) Branch() *branch.Branch {
	return d.branch
}
