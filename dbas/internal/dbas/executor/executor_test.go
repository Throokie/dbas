package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	dbasconfig "reasonix/dbas/internal/dbas/config"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/persistence"
)

func TestCommandRunnerPipesInputToStdin(t *testing.T) {
	cfg := dbasconfig.ExecutorConfig{Type: "command"}
	if runtime.GOOS == "windows" {
		cfg.Command = "cmd"
		cfg.Args = []string{"/C", "more"}
	} else {
		cfg.Command = "cat"
	}
	runner := &CommandRunner{cfg: cfg}

	var out bytes.Buffer
	if err := runner.RunWithOutput(context.Background(), "hello dbas", &out); err != nil {
		t.Fatalf("RunWithOutput() error = %v", err)
	}
	if !strings.Contains(out.String(), "hello dbas") {
		t.Fatalf("output = %q, want piped input", out.String())
	}
}

func TestCommandRunnerRunTaskWritesArtifacts(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	logPath := filepath.Join(root, "execution.log")
	reportPath := filepath.Join(root, "report.json")

	b := branch.NewBranch("branch-command", nil, "root", "branch", experienceslot.DefaultSlot())
	b.OutputPaths.WorkspaceDir = workspace
	b.OutputPaths.LogFile = logPath
	b.OutputPaths.ResultReport = reportPath

	cfg := dbasconfig.ExecutorConfig{Type: "command"}
	if runtime.GOOS == "windows" {
		cfg.Command = "cmd"
		cfg.Args = []string{"/C", "echo workspace=%DBAS_WORKSPACE_DIR% && echo report={report} && echo cwd>cwd.txt"}
	} else {
		cfg.Command = "sh"
		cfg.Args = []string{"-c", "echo workspace=$DBAS_WORKSPACE_DIR; echo report={report}; echo cwd > cwd.txt"}
	}
	runner := &CommandRunner{cfg: cfg}

	if err := runner.RunTask(context.Background(), NewTask(b, "make it so")); err != nil {
		t.Fatalf("RunTask() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "task.json")); err != nil {
		t.Fatalf("task.json missing: %v", err)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logData), "DBAS_EXECUTOR_START") || !strings.Contains(string(logData), workspace) || !strings.Contains(string(logData), reportPath) {
		t.Fatalf("log missing executor evidence:\n%s", string(logData))
	}
	if _, err := os.Stat(filepath.Join(workspace, "cwd.txt")); err != nil {
		t.Fatalf("command did not run in workspace: %v", err)
	}

	var report persistence.ResultReport
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, string(reportData))
	}
	if report.BranchID != "branch-command" || report.Status != "partial" {
		t.Fatalf("report = %#v, want branch-command partial", report)
	}
}
