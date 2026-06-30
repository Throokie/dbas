package kanban

import (
	"context"
	"fmt"
	"os"
	"time"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
)

// Daemon 是 DBAS 的看板守护进程。
// 它不消耗任何 LLM Token，只解析本地分支树元数据并推送状态。
// 它是"纯规则驱动的轻量守护进程"。
type Daemon struct {
	engine       branchQueryer
	notifier     Notifier
	pollInterval time.Duration
	stopCh       chan struct{}
}

// branchQueryer 是分支查询接口（由 BranchEngine 实现）。
type branchQueryer interface {
	GetAllBranches() []*branch.Branch
	ExperienceSlot() *experienceslot.ExperienceSlot
}

// Notifier 是通知接口（由 Feishu/Lark/Console 等实现）。
type Notifier interface {
	Notify(card *KanbanCard) error
}

// KanbanCard 是看板卡片的数据结构。
type KanbanCard struct {
	ProjectID      string         `json:"project_id"`
	Timestamp      int64          `json:"timestamp"`
	ActiveBranches []BranchView   `json:"active_branches"`
	TerminalCount  int            `json:"terminal_count"`
	Slot           *experienceslot.ExperienceSlot `json:"experience_slot"`
}

// BranchView 是分支的看板视图（精简字段，适合卡片展示）。
type BranchView struct {
	ID            string `json:"id"`
	Name          string `json:"name"` // 从 ID 提取的任务名
	State         string `json:"state"`
	Model         string `json:"model"`
	Retries       int    `json:"retries"`
	LogCount      int    `json:"log_count"`
	IsTerminal    bool   `json:"is_terminal"`
	WorkspaceDir  string `json:"workspace_dir"`
}

// NewDaemon 初始化看板守护进程。
func NewDaemon(engine branchQueryer, notifier Notifier, pollInterval time.Duration) *Daemon {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &Daemon{
		engine:       engine,
		notifier:     notifier,
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动守护进程（阻塞直到 Stop 被调用）。
func (d *Daemon) Start(ctx context.Context) {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	// 首次立即推送
	d.push()

	for {
		select {
		case <-ticker.C:
			d.push()
		case <-d.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop 停止守护进程。
func (d *Daemon) Stop() {
	close(d.stopCh)
}

// push 构造当前看板卡片并推送。
func (d *Daemon) push() {
	branches := d.engine.GetAllBranches()
	slot := d.engine.ExperienceSlot()

	var active []BranchView
	terminalCount := 0
	for _, b := range branches {
		view := BranchView{
			ID:           b.BranchID,
			Name:         extractName(b.BranchID),
			State:        string(b.CurrentState),
			Model:        b.ExecutorInfo.Model,
			Retries:      b.Retries,
			LogCount:     len(b.Logs),
			IsTerminal:   b.IsTerminal(),
			WorkspaceDir: b.OutputPaths.WorkspaceDir,
		}
		if b.IsTerminal() {
			terminalCount++
		} else {
			active = append(active, view)
		}
	}

	card := &KanbanCard{
		ProjectID:      slot.ProjectID,
		Timestamp:      time.Now().Unix(),
		ActiveBranches: active,
		TerminalCount:  terminalCount,
		Slot:           slot,
	}

	if d.notifier != nil {
		if err := d.notifier.Notify(card); err != nil {
			// 守护进程不崩溃，只记录错误
			fmt.Fprintf(os.Stderr, "[kanban] notify error: %v\n", err)
		}
	}
}

func extractName(branchID string) string {
	// branch-{hex}-{task-name}
	// 提取最后的任务名部分
	for i := len(branchID) - 1; i >= 0; i-- {
		if branchID[i] == '-' {
			if i > 0 && branchID[i-1] == '-' {
				return branchID[i+1:]
			}
		}
	}
	return branchID
}

// ConsoleNotifier 是打印到控制台的 notifier（调试/测试用）。
type ConsoleNotifier struct{}

// Notify 打印看板卡片到 stdout。
func (c *ConsoleNotifier) Notify(card *KanbanCard) error {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  DBAS Kanban Daemon | Project: %-28s ║\n", card.ProjectID)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Active Branches: %-4d  |  Terminal: %-4d               ║\n", len(card.ActiveBranches), card.TerminalCount)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	for _, b := range card.ActiveBranches {
		stateIcon := "🟡"
		if b.IsTerminal {
			stateIcon = "🟢"
		}
		if b.State == "ERROR" || b.State == "FAIL_RETREAT" {
			stateIcon = "🔴"
		}
		fmt.Printf("║  %s %-20s | %-12s | %-18s | r=%d ║\n",
			stateIcon, b.Name, b.State, b.Model, b.Retries)
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	return nil
}
