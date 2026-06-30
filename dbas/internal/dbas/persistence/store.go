package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"reasonix/dbas/internal/dbas/branch"
)

// BranchStore 负责分支的持久化存储。
// 每个分支在磁盘上有一个独立的目录结构：
//   ./branches/{branch_id}/
//     ├── branch_meta.json   — 分支元数据
//     ├── workspace/          — 执行器的工作目录
//     ├── execution.log      — 执行日志
//     └── report.json        — 结果报告
type BranchStore struct {
	rootDir string
	mu      sync.RWMutex
}

// NewBranchStore 初始化存储，rootDir 是 ./branches/ 的父目录。
func NewBranchStore(rootDir string) *BranchStore {
	return &BranchStore{rootDir: rootDir}
}

// branchDir 返回分支的磁盘目录。
func (s *BranchStore) branchDir(branchID string) string {
	return filepath.Join(s.rootDir, "branches", branchID)
}

// InitBranch 在磁盘上创建分支的目录结构。
func (s *BranchStore) InitBranch(b *branch.Branch) error {
	dir := s.branchDir(b.BranchID)
	workspacePath := filepath.Join(dir, "workspace")
	logPath := filepath.Join(dir, "execution.log")
	reportPath := filepath.Join(dir, "report.json")

	paths := []string{
		dir,
		workspacePath,
	}
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", p, err)
		}
	}

	// 创建空文件
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if err := os.WriteFile(logPath, []byte{}, 0o644); err != nil {
			return fmt.Errorf("create log file: %w", err)
		}
	}

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		emptyReport := ResultReport{CreatedAt: time.Now().UTC()}
		data, _ := json.MarshalIndent(emptyReport, "", "  ")
		if err := os.WriteFile(reportPath, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("create report file: %w", err)
		}
	}

	// 保存 meta
	b.OutputPaths.WorkspaceDir = workspacePath
	b.OutputPaths.LogFile = logPath
	b.OutputPaths.ResultReport = reportPath

	if err := s.SaveBranchMeta(b); err != nil {
		return fmt.Errorf("save meta: %w", err)
	}

	return nil
}

// SaveBranchMeta 将分支元数据持久化到 branch_meta.json。
func (s *BranchStore) SaveBranchMeta(b *branch.Branch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.branchDir(b.BranchID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	metaPath := filepath.Join(dir, "branch_meta.json")
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal branch meta: %w", err)
	}
	data = append(data, '\n')

	// 原子写入：先写临时文件，再重命名
	tmpPath := metaPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write tmp meta: %w", err)
	}
	if err := os.Rename(tmpPath, metaPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename meta: %w", err)
	}
	return nil
}

// LoadBranchMeta 从磁盘加载分支元数据。
func (s *BranchStore) LoadBranchMeta(branchID string) (*branch.Branch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metaPath := filepath.Join(s.branchDir(branchID), "branch_meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read meta: %w", err)
	}

	var b branch.Branch
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("unmarshal meta: %w", err)
	}

	// 重建 FSM：从持久化的 CurrentState 恢复状态机
	b.RebuildFSM()
	return &b, nil
}

// AppendLog 追加一行日志到 execution.log。
func (s *BranchStore) AppendLog(branchID string, entry string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logPath := filepath.Join(s.branchDir(branchID), "execution.log")
	line := fmt.Sprintf("%s\n", entry)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("append log: %w", err)
	}
	return nil
}

// WriteReport 写入结果报告。
func (s *BranchStore) WriteReport(branchID string, report ResultReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reportPath := filepath.Join(s.branchDir(branchID), "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// ReadReport 读取结果报告。
func (s *BranchStore) ReadReport(branchID string) (*ResultReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reportPath := filepath.Join(s.branchDir(branchID), "report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("read report: %w", err)
	}
	var report ResultReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("unmarshal report: %w", err)
	}
	return &report, nil
}

// ListBranches 返回磁盘上所有已持久化的分支 ID。
func (s *BranchStore) ListBranches() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	branchesDir := filepath.Join(s.rootDir, "branches")
	entries, err := os.ReadDir(branchesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// ResultReport 是执行器的标准化结果报告。
type ResultReport struct {
	TaskID      string            `json:"task_id"`
	BranchID    string            `json:"branch_id"`
	Status      string            `json:"status"` // "success" | "failure" | "partial"
	Summary     string            `json:"summary"`
	Files       []string          `json:"files"`
	Commands    []string          `json:"commands"`
	TestResults map[string]string `json:"test_results,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}
