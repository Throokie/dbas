package branch

import (
	"crypto/sha256"
	"fmt"
	"time"

	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
)

// Branch 是 DBAS 的核心分支实体，对应 Reasonix 的 BranchMeta 并扩展 DBAS 专属字段。
type Branch struct {
	BranchID       string  `json:"branch_id"`
	ParentBranchID *string `json:"parent_branch_id,omitempty"`

	// HashingChain 对应 DBAS 架构手册的 Zone 1/2 哈希链。
	HashingChain HashingChain `json:"hashing_chain"`

	// CurrentState 由 FSM 控制，严禁外部直接修改。
	CurrentState fsm.State `json:"current_state"`

	// ExecutorInfo 记录当前执行器的模型配置。
	ExecutorInfo ExecutorInfo `json:"executor_info"`

	// OutputPaths 定义分支的本地工作目录和输出文件路径。
	OutputPaths OutputPaths `json:"output_paths"`

	// Logs 是执行历史日志（按时间顺序）。
	Logs []string `json:"logs"`

	// Retries 是当前状态的重试计数（用于 escalation 判断）。
	Retries int `json:"retries"`

	// CreatedAt 是分支创建时间。
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 是最后更新时间。
	UpdatedAt time.Time `json:"updated_at"`

	// --- 非持久化运行时字段 ---

	// fsm 是绑定的状态机实例（不直接序列化，从 CurrentState 重建）。
	fsm *fsm.StateMachine

	// slot 是绑定的经验槽规则。
	slot *experienceslot.ExperienceSlot
}

// HashingChain 对应 DBAS 三区缓存架构的哈希链。
type HashingChain struct {
	// RootHash = SHA-256(Zone 1: System Prompt + 字母排序的 Tool Schema)
	RootHash string `json:"root_hash"`
	// BranchHash = SHA-256(Zone 2: Shared Context + Parent Context)
	BranchHash string `json:"branch_hash"`
}

// ExecutorInfo 记录执行器配置。
type ExecutorInfo struct {
	Model           string `json:"model"`
	AllocatedTokens int    `json:"allocated_tokens"`
}

// OutputPaths 定义分支的本地文件路径。
type OutputPaths struct {
	WorkspaceDir string `json:"workspace_dir"`
	LogFile      string `json:"log_file"`
	ResultReport string `json:"result_report"`
}

// NewBranch 创建一个分支，初始化 FSM 和经验槽。
func NewBranch(id string, parentID *string, rootHash, branchHash string, slot *experienceslot.ExperienceSlot) *Branch {
	now := time.Now().UTC()
	b := &Branch{
		BranchID:       id,
		ParentBranchID: parentID,
		HashingChain: HashingChain{
			RootHash:   rootHash,
			BranchHash: branchHash,
		},
		CurrentState: fsm.StateINIT,
		ExecutorInfo: ExecutorInfo{
			Model:           "default-agent",
			AllocatedTokens: 8192,
		},
		OutputPaths: OutputPaths{
			WorkspaceDir: fmt.Sprintf("./branches/%s/workspace", id),
			LogFile:      fmt.Sprintf("./branches/%s/execution.log", id),
			ResultReport: fmt.Sprintf("./branches/%s/report.json", id),
		},
		Logs:      []string{fmt.Sprintf("[%s] Branch created: %s", now.Format(time.RFC3339), id)},
		Retries:   0,
		CreatedAt: now,
		UpdatedAt: now,
		fsm:       fsm.NewStateMachine(),
		slot:      slot,
	}
	return b
}

// FSM 返回分支绑定的状态机，如果未初始化则自动重建。
func (b *Branch) FSM() *fsm.StateMachine {
	if b.fsm == nil {
		b.fsm = fsm.NewStateMachineWithState(b.CurrentState)
	}
	return b.fsm
}

// RebuildFSM 从持久化的 CurrentState 重建状态机。
func (b *Branch) RebuildFSM() {
	b.fsm = fsm.NewStateMachineWithState(b.CurrentState)
}

// Slot 返回分支绑定的经验槽。
func (b *Branch) Slot() *experienceslot.ExperienceSlot {
	return b.slot
}

// Transition 通过 FSM 执行状态转换，并记录日志。
func (b *Branch) Transition(target fsm.State, reason string) error {
	if err := b.FSM().Transition(target, reason); err != nil {
		b.Logs = append(b.Logs, fmt.Sprintf("[FSM_REJECT] %v", err))
		b.UpdatedAt = time.Now().UTC()
		return err
	}
	b.CurrentState = b.FSM().Current()
	b.Logs = append(b.Logs, fmt.Sprintf("[STATE_CHANGE] %s -> %s | %s", target, b.CurrentState, reason))
	b.UpdatedAt = time.Now().UTC()
	return nil
}

// IncrementRetries 增加重试计数。
func (b *Branch) IncrementRetries() {
	b.Retries++
	b.Logs = append(b.Logs, fmt.Sprintf("[RETRY] attempt %d/%d", b.Retries, b.slot.Rules.EscalationThreshold))
	b.UpdatedAt = time.Now().UTC()
}

// ResetRetries 重置重试计数（通常在升级后调用）。
func (b *Branch) ResetRetries() {
	b.Retries = 0
	b.Logs = append(b.Logs, "[RESET] retries cleared after escalation")
	b.UpdatedAt = time.Now().UTC()
}

// Escalate 将执行器升级为更强模型并重置重试计数。
func (b *Branch) Escalate() {
	b.ExecutorInfo.Model = "default-agent-fallback"
	b.Logs = append(b.Logs, fmt.Sprintf("[ESCALATE] Executor upgraded to %s after %d failures", b.ExecutorInfo.Model, b.Retries))
	b.ResetRetries()
	b.UpdatedAt = time.Now().UTC()
}

// IsTerminal 返回分支是否已到达终止状态。
func (b *Branch) IsTerminal() bool {
	return b.FSM().IsTerminal()
}

// IsRunning 返回分支是否正在执行中。
func (b *Branch) IsRunning() bool {
	return b.FSM().IsRunning()
}

// ComputeRootHash 计算 Zone 1 的 SHA-256 哈希。
// 实际实现中需要传入 System Prompt 和排序后的 Tool Schema。
func ComputeRootHash(systemPrompt string, toolSchemaJSON string) string {
	h := sha256.New()
	h.Write([]byte(systemPrompt))
	h.Write([]byte(toolSchemaJSON))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ComputeBranchHash 计算 Zone 2 的 SHA-256 哈希。
// 实际实现中需要传入共享上下文内容。
func ComputeBranchHash(parentHash string, contextContent string) string {
	h := sha256.New()
	if parentHash != "" {
		h.Write([]byte(parentHash))
	}
	h.Write([]byte(contextContent))
	return fmt.Sprintf("%x", h.Sum(nil))
}
