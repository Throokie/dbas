package persistence

import (
	"fmt"
	"os"
	"path/filepath"
)

// WorkspaceManager 管理分支的工作目录，提供安全的文件操作。
type WorkspaceManager struct {
	store *BranchStore
}

// NewWorkspaceManager 初始化工作区管理器。
func NewWorkspaceManager(store *BranchStore) *WorkspaceManager {
	return &WorkspaceManager{store: store}
}

// WorkspaceDir 返回分支的工作目录路径。
func (w *WorkspaceManager) WorkspaceDir(branchID string) string {
	return filepath.Join(w.store.branchDir(branchID), "workspace")
}

// WriteFile 在分支 workspace 中安全写入文件。
// 拒绝任何路径逃逸尝试（如 ../etc/passwd）。
func (w *WorkspaceManager) WriteFile(branchID string, relPath string, content []byte) error {
	abs, err := w.safePath(branchID, relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(abs, content, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// ReadFile 读取分支 workspace 中的文件。
func (w *WorkspaceManager) ReadFile(branchID string, relPath string) ([]byte, error) {
	abs, err := w.safePath(branchID, relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

// ListFiles 递归列出分支 workspace 中的所有文件。
func (w *WorkspaceManager) ListFiles(branchID string) ([]string, error) {
	dir := w.WorkspaceDir(branchID)
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		out = append(out, rel)
		return nil
	})
	return out, err
}

// DeleteFile 删除分支 workspace 中的文件。
func (w *WorkspaceManager) DeleteFile(branchID string, relPath string) error {
	abs, err := w.safePath(branchID, relPath)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}

// safePath 确保相对路径不会逃逸出 workspace。
func (w *WorkspaceManager) safePath(branchID string, relPath string) (string, error) {
	dir := w.WorkspaceDir(branchID)
	abs := filepath.Join(dir, relPath)
	abs = filepath.Clean(abs)

	rel, err := filepath.Rel(dir, abs)
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("path %q escapes workspace", relPath)
	}
	return abs, nil
}

// ClearWorkspace 清空分支的工作目录（危险操作，谨慎使用）。
func (w *WorkspaceManager) ClearWorkspace(branchID string) error {
	dir := w.WorkspaceDir(branchID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}
