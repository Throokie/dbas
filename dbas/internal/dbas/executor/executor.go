package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"reasonix/dbas/internal/dbas/branch"
	dbasconfig "reasonix/dbas/internal/dbas/config"
	"reasonix/dbas/internal/dbas/persistence"
	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

type Runner interface {
	Run(ctx context.Context, input string) error
}

type TaskRunner interface {
	RunTask(ctx context.Context, task Task) error
}

type Task struct {
	TaskID     string              `json:"task_id"`
	BranchID   string              `json:"branch_id"`
	Input      string              `json:"input"`
	Attempt    int                 `json:"attempt"`
	Executor   branch.ExecutorInfo `json:"executor"`
	RootHash   string              `json:"root_hash"`
	BranchHash string              `json:"branch_hash"`
	Paths      branch.OutputPaths  `json:"paths"`
	CreatedAt  time.Time           `json:"created_at"`
}

func NewTask(b *branch.Branch, input string) Task {
	return Task{
		TaskID:     b.BranchID,
		BranchID:   b.BranchID,
		Input:      input,
		Attempt:    b.Retries + 1,
		Executor:   b.ExecutorInfo,
		RootHash:   b.HashingChain.RootHash,
		BranchHash: b.HashingChain.BranchHash,
		Paths:      b.OutputPaths,
		CreatedAt:  time.Now().UTC(),
	}
}

func New(cfg dbasconfig.ExecutorConfig, prov provider.Provider, registry *tool.Registry, handoffPrompt string, contextWindow int) (Runner, error) {
	switch cfg.Type {
	case "", "reasonix-native":
		if contextWindow <= 0 {
			contextWindow = 128000
		}
		session := agent.NewSession(handoffPrompt)
		return agent.New(prov, registry, session, agent.Options{
			MaxSteps:      100,
			Temperature:   0.0,
			ContextWindow: contextWindow,
		}, event.Discard), nil
	case "command":
		return &CommandRunner{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported executor type %q", cfg.Type)
	}
}

type CommandRunner struct {
	cfg dbasconfig.ExecutorConfig
}

func (r *CommandRunner) Run(ctx context.Context, input string) error {
	var output bytes.Buffer
	if err := r.run(ctx, input, "", r.cfg.Args, r.cfg.Env, nil, &output); err != nil {
		return fmt.Errorf("command executor failed: %w\n%s", err, limitString(output.String(), 8192))
	}
	return nil
}

func (r *CommandRunner) RunTask(ctx context.Context, task Task) error {
	if err := ensureTaskFiles(task); err != nil {
		return err
	}

	logFile, err := os.OpenFile(task.Paths.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open executor log: %w", err)
	}
	defer logFile.Close()

	stdin, err := r.stdinForTask(task)
	if err != nil {
		return err
	}
	fmt.Fprintf(logFile, "\n[DBAS_EXECUTOR_START] branch=%s attempt=%d command=%s args=%v\n", task.BranchID, task.Attempt, r.cfg.Command, r.cfg.Args)
	err = r.run(ctx, stdin, r.workdirForTask(task), expandArgs(r.cfg.Args, task), expandEnv(r.cfg.Env, task), taskEnv(task), logFile)
	if err != nil {
		fmt.Fprintf(logFile, "[DBAS_EXECUTOR_ERROR] %v\n", err)
		_ = writeFallbackReport(task, "failure", "command executor failed", []string{err.Error()})
		return fmt.Errorf("command executor failed: %w", err)
	}
	fmt.Fprintf(logFile, "[DBAS_EXECUTOR_DONE] branch=%s attempt=%d\n", task.BranchID, task.Attempt)
	return ensureReport(task)
}

func (r *CommandRunner) RunWithOutput(ctx context.Context, input string, out io.Writer) error {
	return r.run(ctx, input, "", r.cfg.Args, r.cfg.Env, nil, out)
}

func (r *CommandRunner) run(ctx context.Context, input, workdir string, args []string, env map[string]string, extraEnv []string, out io.Writer) error {
	if workdir == "" {
		workdir = r.cfg.Workdir
	}
	cmd := exec.CommandContext(ctx, r.cfg.Command, args...)
	cmd.Dir = workdir
	cmd.Env = append(append(os.Environ(), flattenEnv(env)...), extraEnv...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

func (r *CommandRunner) workdirForTask(task Task) string {
	if r.cfg.Workdir == "" {
		return task.Paths.WorkspaceDir
	}
	return expandTaskPlaceholders(r.cfg.Workdir, task)
}

func expandArgs(args []string, task Task) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = expandTaskPlaceholders(arg, task)
	}
	return out
}

func expandEnv(env map[string]string, task Task) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = expandTaskPlaceholders(v, task)
	}
	return out
}

func expandTaskPlaceholders(value string, task Task) string {
	replacements := map[string]string{
		"{task_id}":     task.TaskID,
		"{branch_id}":   task.BranchID,
		"{attempt}":     fmt.Sprintf("%d", task.Attempt),
		"{workspace}":   task.Paths.WorkspaceDir,
		"{report}":      task.Paths.ResultReport,
		"{log}":         task.Paths.LogFile,
		"{task_file}":   filepath.Join(filepath.Dir(task.Paths.ResultReport), "task.json"),
		"{root_hash}":   task.RootHash,
		"{branch_hash}": task.BranchHash,
	}
	out := value
	for from, to := range replacements {
		out = strings.ReplaceAll(out, from, to)
	}
	return out
}

func (r *CommandRunner) stdinForTask(task Task) (string, error) {
	if r.cfg.StdinFormat == "json" {
		data, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal executor task: %w", err)
		}
		return string(data) + "\n", nil
	}
	return formatTextTask(task), nil
}

func formatTextTask(task Task) string {
	return fmt.Sprintf(`# DBAS Executor Task

Branch: %s
Attempt: %d
Workspace: %s
Report file: %s
Log file: %s

Write all file changes under the workspace directory. When finished, write a JSON report to the report file with this shape:
{
  "task_id": "%s",
  "branch_id": "%s",
  "status": "success|failure|partial",
  "summary": "short result summary",
  "files": ["relative/path"],
  "commands": ["command output summary"],
  "test_results": {"name": "result"},
  "errors": []
}

## Task
%s
`, task.BranchID, task.Attempt, task.Paths.WorkspaceDir, task.Paths.ResultReport, task.Paths.LogFile, task.TaskID, task.BranchID, task.Input)
}

func ensureTaskFiles(task Task) error {
	if task.Paths.WorkspaceDir == "" || task.Paths.LogFile == "" || task.Paths.ResultReport == "" {
		return fmt.Errorf("executor task paths are incomplete")
	}
	if err := os.MkdirAll(task.Paths.WorkspaceDir, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	path := filepath.Join(filepath.Dir(task.Paths.ResultReport), "task.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write task.json: %w", err)
	}
	return nil
}

func taskEnv(task Task) []string {
	return []string{
		"DBAS_TASK_ID=" + task.TaskID,
		"DBAS_BRANCH_ID=" + task.BranchID,
		"DBAS_ATTEMPT=" + fmt.Sprintf("%d", task.Attempt),
		"DBAS_WORKSPACE_DIR=" + task.Paths.WorkspaceDir,
		"DBAS_REPORT_FILE=" + task.Paths.ResultReport,
		"DBAS_LOG_FILE=" + task.Paths.LogFile,
		"DBAS_TASK_FILE=" + filepath.Join(filepath.Dir(task.Paths.ResultReport), "task.json"),
	}
}

func ensureReport(task Task) error {
	data, err := os.ReadFile(task.Paths.ResultReport)
	if err == nil && strings.TrimSpace(string(data)) != "" {
		var report persistence.ResultReport
		if json.Unmarshal(data, &report) == nil && report.Status != "" {
			return nil
		}
	}
	return writeFallbackReport(task, "partial", "executor completed but did not write a DBAS report", nil)
}

func writeFallbackReport(task Task, status, summary string, errors []string) error {
	report := persistence.ResultReport{
		TaskID:    task.TaskID,
		BranchID:  task.BranchID,
		Status:    status,
		Summary:   summary,
		Files:     listWorkspaceFiles(task.Paths.WorkspaceDir),
		Commands:  []string{"command executor"},
		Errors:    errors,
		CreatedAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal fallback report: %w", err)
	}
	if err := os.WriteFile(task.Paths.ResultReport, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write fallback report: %w", err)
	}
	return nil
}

func listWorkspaceFiles(workspace string) []string {
	var files []string
	_ = filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(workspace, path)
		if err == nil {
			files = append(files, rel)
		}
		return nil
	})
	return files
}

func flattenEnv(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func limitString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n[output truncated]"
}
