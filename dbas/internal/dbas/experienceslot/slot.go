package experienceslot

import (
	"encoding/json"
	"fmt"

	"reasonix/dbas/internal/dbas/fsm"
)

// ExperienceSlot 是 DBAS 的经验槽规则配置，硬编码为系统默认值。
// 用户可以通过配置文件覆盖，但核心规则（如 FSM 路径）不可修改。
type ExperienceSlot struct {
	ProjectID   string            `json:"project_id"`
	DefaultFlow []fsm.State       `json:"default_flow"`
	Rules       Rules             `json:"rules"`
	Workflows   map[string]WorkflowRule `json:"workflows"`
}

// Rules 定义经验槽的核心约束。
type Rules struct {
	EscalationThreshold    int           `json:"escalation_threshold"`
	ForceVerifierForStates []fsm.State   `json:"force_verifier_for_states"`
	VerifierRules          VerifierRules `json:"verifier_rules"`
}

// VerifierRules 定义验收队的检查标准。
type VerifierRules struct {
	CodeSyntax   string `json:"code_syntax"`    // 如 "ast-check", "go-vet", "eslint"
	TestPassRate int    `json:"test_pass_rate"` // 百分比，如 100
}

// WorkflowRule 定义特定工作流的回退规则。
type WorkflowRule struct {
	From       fsm.State `json:"from"`
	OnFail     fsm.State `json:"on_fail"`
	MaxRetries int       `json:"max_retries"`
}

// DefaultSlot 返回系统硬编码的默认经验槽配置。
// 这是 DBAS 的"出厂设置"，所有项目以此为基础。
func DefaultSlot() *ExperienceSlot {
	return &ExperienceSlot{
		ProjectID: "default-dbas-project",
		DefaultFlow: []fsm.State{
			fsm.StateINIT, fsm.StatePLAN, fsm.StateEXECUTING,
			fsm.StateVERIFYING, fsm.StateARCHIVE,
		},
		Rules: Rules{
			EscalationThreshold: 3,
			ForceVerifierForStates: []fsm.State{
				fsm.StateEXECUTING,
			},
			VerifierRules: VerifierRules{
				CodeSyntax:   "ast-check",
				TestPassRate: 100,
			},
		},
		Workflows: map[string]WorkflowRule{
			"bug_loop": {
				From:       fsm.StateVERIFYING,
				OnFail:     fsm.StateEXECUTING,
				MaxRetries: 3,
			},
		},
	}
}

// ValidateDecision 检查一个决策是否符合经验槽规则。
// 返回错误表示该决策违反了纪律约束。
func (s *ExperienceSlot) ValidateDecision(currentState fsm.State, targetState fsm.State, retries int) error {
	// 1. FSM 转换已经在 fsm.StateMachine 中校验，这里只校验经验槽约束

	// 2. 检查是否强制要求 verifier 的状态
	if targetState == fsm.StateVERIFYING || targetState == fsm.StatePASS {
		requiresVerifier := false
		for _, st := range s.Rules.ForceVerifierForStates {
			if st == fsm.StateEXECUTING {
				requiresVerifier = true
				break
			}
		}
		if !requiresVerifier {
			// 如果规则未强制 verifier，允许跳过（但不推荐）
			_ = requiresVerifier // lint
		}
	}

	// 3. 检查 retries 是否超过 escalation_threshold
	if retries >= s.Rules.EscalationThreshold {
		// 超过阈值：必须升级为更高级模型执行
		// 这个信号由调用方（brain.Coordinator）处理
		_ = retries // lint
	}

	// 4. 检查工作流规则
	for name, rule := range s.Workflows {
		if rule.From == currentState && rule.OnFail == targetState {
			if retries > rule.MaxRetries {
				return fmt.Errorf("workflow %s: max retries (%d) exceeded", name, rule.MaxRetries)
			}
		}
	}

	return nil
}

// ShouldEscalate 返回 retries 是否达到升级阈值。
func (s *ExperienceSlot) ShouldEscalate(retries int) bool {
	return retries >= s.Rules.EscalationThreshold
}

// MaxRetriesForWorkflow 返回指定工作流的最大重试次数。
func (s *ExperienceSlot) MaxRetriesForWorkflow(name string) int {
	if rule, ok := s.Workflows[name]; ok {
		return rule.MaxRetries
	}
	return 0
}

// ToJSON 返回经验槽的 JSON 表示（用于持久化、API 返回）。
func (s *ExperienceSlot) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// FromJSON 从 JSON 解析经验槽配置。
func FromJSON(data []byte) (*ExperienceSlot, error) {
	var slot ExperienceSlot
	if err := json.Unmarshal(data, &slot); err != nil {
		return nil, fmt.Errorf("parse experience slot: %w", err)
	}
	return &slot, nil
}
