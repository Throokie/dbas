package branch

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
)

// BranchEngine 管理所有分支的生命周期。
// 它是 DBAS 的"分支树根节点"，替代 Reasonix 的 ListBranches。
type BranchEngine struct {
	mu       sync.RWMutex
	branches map[string]*Branch
	slot     *experienceslot.ExperienceSlot

	// rootHash 是 Zone 1 的哈希，所有分支共享。
	rootHash string
}

// NewEngine 初始化分支引擎。
func NewEngine(rootHash string, slot *experienceslot.ExperienceSlot) *BranchEngine {
	if slot == nil {
		slot = experienceslot.DefaultSlot()
	}
	return &BranchEngine{
		branches: make(map[string]*Branch),
		slot:     slot,
		rootHash: rootHash,
	}
}

// CreateBranch 创建一个新分支。
// parentID 为 nil 表示根分支。
func (e *BranchEngine) CreateBranch(taskName string, parentID *string) (*Branch, error) {
	id := e.generateBranchID(taskName)

	var parentHash string
	if parentID != nil {
		if parent, ok := e.branches[*parentID]; ok {
			parentHash = parent.HashingChain.BranchHash
		}
	}

	// Zone 2 哈希：基于父分支哈希 + 任务名（简化实现）
	branchHash := ComputeBranchHash(parentHash, taskName)

	b := NewBranch(id, parentID, e.rootHash, branchHash, e.slot)

	e.mu.Lock()
	defer e.mu.Unlock()
	e.branches[id] = b
	return b, nil
}

// GetBranch 按 ID 查询分支。
func (e *BranchEngine) GetBranch(id string) (*Branch, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	b, ok := e.branches[id]
	return b, ok
}

// GetAllBranches 返回所有分支（只读副本）。
func (e *BranchEngine) GetAllBranches() []*Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*Branch, 0, len(e.branches))
	for _, b := range e.branches {
		out = append(out, b)
	}
	return out
}

// GetActiveBranches 返回非终止状态的分支。
func (e *BranchEngine) GetActiveBranches() []*Branch {
	var out []*Branch
	for _, b := range e.GetAllBranches() {
		if !b.IsTerminal() {
			out = append(out, b)
		}
	}
	return out
}

// ResetBranch 将分支重置到 INIT 状态。
func (e *BranchEngine) ResetBranch(id string) error {
	b, ok := e.GetBranch(id)
	if !ok {
		return fmt.Errorf("branch not found: %s", id)
	}
	if b.IsTerminal() {
		// 即使已终止，也允许重置（手动干预）
	}
	b.CurrentState = fsm.StateINIT
	b.Retries = 0
	b.ExecutorInfo.Model = "default-agent"
	b.Logs = append(b.Logs, fmt.Sprintf("[MANUAL_RESET] Branch reset to INIT by user/command"))
	return nil
}

// ExperienceSlot 返回引擎绑定的经验槽。
func (e *BranchEngine) ExperienceSlot() *experienceslot.ExperienceSlot {
	return e.slot
}

// RootHash 返回 Zone 1 根哈希。
func (e *BranchEngine) RootHash() string {
	return e.rootHash
}

// generateBranchID 生成唯一的分支 ID。
func (e *BranchEngine) generateBranchID(taskName string) string {
	b := make([]byte, 4)
	rand.Read(b)
	suffix := hex.EncodeToString(b)
	return fmt.Sprintf("branch-%s-%s", suffix, sanitizeTaskName(taskName))
}

func sanitizeTaskName(name string) string {
	// 将空格替换为 -，去除非字母数字字符
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == ' ', r == '-', r == '_':
			out = append(out, '-')
		}
	}
	return string(out)
}
