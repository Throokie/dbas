package persistence

import (
	"path/filepath"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
)

func TestInitBranchRewritesOutputPathsUnderStoreRoot(t *testing.T) {
	root := t.TempDir()
	store := NewBranchStore(root)
	b := branch.NewBranch("branch-test", nil, "root", "branch", experienceslot.DefaultSlot())

	if err := store.InitBranch(b); err != nil {
		t.Fatalf("InitBranch() error = %v", err)
	}

	wantWorkspace := filepath.Join(root, "branches", "branch-test", "workspace")
	wantLog := filepath.Join(root, "branches", "branch-test", "execution.log")
	wantReport := filepath.Join(root, "branches", "branch-test", "report.json")
	if b.OutputPaths.WorkspaceDir != wantWorkspace || b.OutputPaths.LogFile != wantLog || b.OutputPaths.ResultReport != wantReport {
		t.Fatalf("OutputPaths = %#v, want workspace=%q log=%q report=%q", b.OutputPaths, wantWorkspace, wantLog, wantReport)
	}

	loaded, err := store.LoadBranchMeta("branch-test")
	if err != nil {
		t.Fatalf("LoadBranchMeta() error = %v", err)
	}
	if loaded.OutputPaths.WorkspaceDir != wantWorkspace {
		t.Fatalf("persisted workspace = %q, want %q", loaded.OutputPaths.WorkspaceDir, wantWorkspace)
	}
}
