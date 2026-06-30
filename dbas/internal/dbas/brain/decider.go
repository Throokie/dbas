package brain
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/internal/provider"
)
// DeciderProposal 鏄?Qwen 35B 涓昏剳杈撳嚭鐨勬彁妗?JSON銆?
type DeciderProposal struct {
	TaskID         string      `json:"task_id"`
	TargetState    fsm.State   `json:"target_state"`
	ExecutionPlan  string      `json:"execution_plan"`
	ToolsNeeded    []string    `json:"tools_needed"`
	ExpectedOutput string      `json:"expected_output"`
	Confidence     float64     `json:"confidence"`
}
// Decider 鏄?Qwen 35B 涓昏剳锛岃礋璐ｆ彁鍑轰换鍔℃墽琛屾柟妗堛€?
// 瀹冪殑鑳藉姏琚弗鏍奸檺鍒朵负锛氬喅绛栥€佽鍒掋€佽矾鐢便€佹彁鍙栫己闄风壒寰併€?
// 瀹冪粷涓嶆墽琛屼换浣曞姩浣滐紝涔熶笉鍐欑敓浜т唬鐮併€?
type Decider struct {
	systemPrompt string
	// provider 鏄繛鎺?Qwen 35B 鐨勬帴鍙ｏ紙澶嶇敤 Reasonix provider.Provider锛?
	provider provider.Provider
}
// NewDecider 鍒濆鍖栦富鑴戙€?
func NewDecider(provider provider.Provider) *Decider {
	return &Decider{
		provider:     provider,
		systemPrompt: DeciderSystemPrompt,
	}
}
// Propose 鎺ユ敹浠诲姟杈撳叆锛岃緭鍑?DeciderProposal JSON銆?
func (d *Decider) Propose(ctx context.Context, taskInput string, b *branch.Branch) (*DeciderProposal, error) {
	prompt := d.buildProposalPrompt(taskInput, b)
	req := provider.Request{
		Messages: []provider.Message{
			{Role: "system", Content: d.systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		MaxTokens:   4096,
	}
	ch, err := d.provider.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("decider stream error: %w", err)
	}
	var raw strings.Builder
	for chunk := range ch {
		switch chunk.Type {
		case provider.ChunkText:
			raw.WriteString(chunk.Text)
		case provider.ChunkError:
			return nil, fmt.Errorf("decider error: %w", chunk.Err)
		}
	}
	// 瑙ｆ瀽 JSON 鎻愭
	proposal, err := parseProposal(raw.String())
	if err != nil {
		return nil, fmt.Errorf("decider output parse error: %w", err)
	}
	// 楠岃瘉鎻愭鍩烘湰缁撴瀯
	if proposal.TaskID == "" {
		proposal.TaskID = b.BranchID
	}
	if proposal.TargetState == "" {
		proposal.TargetState = fsm.StateEXECUTING
	}
	return proposal, nil
}
func (d *Decider) buildProposalPrompt(taskInput string, b *branch.Branch) string {
	return fmt.Sprintf(`
# Task Input
%s
# Branch Context
- Branch ID: %s
- Current State: %s
- Retries: %d
- Executor Model: %s
- Experience Slot Rules:
  - Escalation Threshold: %d
  - Verifier Syntax: %s
  - Required Test Pass Rate: %d%%
# Output Format
Respond ONLY with a valid JSON object matching this schema:
{
  "task_id": "string",
  "target_state": "EXECUTING|VERIFYING|ARCHIVE",
  "execution_plan": "string (concise, ordered steps)",
  "tools_needed": ["tool_name_1", "tool_name_2"],
  "expected_output": "string (what success looks like)",
  "confidence": 0.0-1.0
}
**YOU ARE A PURE DECIDER. DO NOT EXECUTE ANY ACTIONS OR WRITE ANY PRODUCTION CODE.**
`, taskInput, b.BranchID, b.CurrentState, b.Retries, b.ExecutorInfo.Model,
		b.Slot().Rules.EscalationThreshold, b.Slot().Rules.VerifierRules.CodeSyntax,
		b.Slot().Rules.VerifierRules.TestPassRate)
}
func parseProposal(raw string) (*DeciderProposal, error) {
	jsonStr := extractJSON(raw)
	var p DeciderProposal
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return nil, err
	}
	return &p, nil
}
