package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"

	"reasonix/internal/provider"
)

// ReviewerDecision 是副脑（Reviewer）输出的审核决策 JSON。
type ReviewerDecision struct {
	Approved       bool      `json:"approved"`
	TargetState    fsm.State `json:"target_state"`
	ExecutionInput string    `json:"execution_input"`
	ReviewNotes    string    `json:"review_notes"`
	Modifications  []string  `json:"modifications,omitempty"`
}

// Reviewer 是审核大脑，负责交叉审核主脑提案。
// 它对照经验槽规则进行验证，输出最终决策。
type Reviewer struct {
	systemPrompt string
	provider     provider.Provider
}

// NewReviewer 初始化副脑。
func NewReviewer(provider provider.Provider) *Reviewer {
	return &Reviewer{
		provider:     provider,
		systemPrompt: ReviewerSystemPrompt,
	}
}

// Review 审核主脑提案，输出最终决策。
func (r *Reviewer) Review(ctx context.Context, proposal *DeciderProposal, b *branch.Branch, slot *experienceslot.ExperienceSlot) (*ReviewerDecision, error) {
	prompt := r.buildReviewPrompt(proposal, b, slot)

	req := provider.Request{
		Messages: []provider.Message{
			{Role: "system", Content: r.systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		MaxTokens:   4096,
	}

	ch, err := r.provider.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("reviewer stream error: %w", err)
	}

	var raw strings.Builder
	for chunk := range ch {
		switch chunk.Type {
		case provider.ChunkText:
			raw.WriteString(chunk.Text)
		case provider.ChunkError:
			return nil, fmt.Errorf("reviewer error: %w", chunk.Err)
		}
	}

	decision, err := parseDecision(raw.String())
	if err != nil {
		return nil, fmt.Errorf("reviewer output parse error: %w", err)
	}

	// 如果审核不通过，强制回到 PLAN 状态
	if !decision.Approved {
		decision.TargetState = fsm.StatePLAN
		decision.ExecutionInput = "REJECTED: " + decision.ReviewNotes
	}

	return decision, nil
}

func (r *Reviewer) buildReviewPrompt(proposal *DeciderProposal, b *branch.Branch, slot *experienceslot.ExperienceSlot) string {
	proposalJSON, _ := json.MarshalIndent(proposal, "", "  ")

	return fmt.Sprintf(`
# Review Request

## Proposal from Decider Brain
%s

## Branch Context
- Branch ID: %s
- Current State: %s
- Retries: %d
- Executor: %s

## Experience Slot Rules
- Escalation Threshold: %d
- Force Verifier For States: %v
- Verifier Code Syntax: %s
- Required Test Pass Rate: %d%%

## Workflows
%s

# Review Instructions
1. Verify the proposal respects the FSM transition rules.
2. Check that the target state is allowed from the current state.
3. Verify the execution plan does not violate the experience slot constraints.
4. If the proposal is acceptable, approve it and refine the execution_input.
5. If not, reject it with detailed review notes and suggest modifications.

# Output Format
Respond ONLY with a valid JSON object:
{
  "approved": true|false,
  "target_state": "EXECUTING|VERIFYING|ARCHIVE|PLAN",
  "execution_input": "string (the final instruction to send to the executor)",
  "review_notes": "string (summary of review)",
  "modifications": ["suggestion 1", "suggestion 2"]
}

**YOU ARE A REVIEWER. VERIFY THE DECISION AGAINST THE EXPERIENCE SLOT RULES. OUTPUT ONLY VALIDATED JSON.**
`, string(proposalJSON), b.BranchID, b.CurrentState, b.Retries, b.ExecutorInfo.Model,
		slot.Rules.EscalationThreshold, slot.Rules.ForceVerifierForStates,
		slot.Rules.VerifierRules.CodeSyntax, slot.Rules.VerifierRules.TestPassRate,
		formatWorkflows(slot.Workflows))
}

func parseDecision(raw string) (*ReviewerDecision, error) {
	jsonStr := extractJSON(raw)
	var d ReviewerDecision
	if err := json.Unmarshal([]byte(jsonStr), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func formatWorkflows(w map[string]experienceslot.WorkflowRule) string {
	if len(w) == 0 {
		return "None"
	}
	var out strings.Builder
	for name, rule := range w {
		fmt.Fprintf(&out, "- %s: %s -> %s (max retries: %d)\n", name, rule.From, rule.OnFail, rule.MaxRetries)
	}
	return out.String()
}

// extractJSON 从文本中提取 JSON 块。
func extractJSON(raw string) string {
	start := strings.Index(raw, "```json")
	if start >= 0 {
		end := strings.Index(raw[start:], "```")
		if end > 7 {
			return strings.TrimSpace(raw[start+7 : start+end])
		}
	}
	start = strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}
