// Package demo provides an offline demo mode for DBAS.
// It uses mock providers and simulated executors to demonstrate
// the full FSM governance lifecycle without requiring any API keys.
//
// When --demo is passed, DBAS starts with:
//   - 5 demo branches (happy path, retry, error, parallel, skills)
//   - 13 domain skills loaded and visible in the dashboard
//   - 3 built-in extensions registered
//   - A self-contained offline experience — no Docker, no API keys
package demo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"reasonix/dbas/internal/dbas/brain"
	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/dbas/internal/dbas/persistence"
	"reasonix/dbas/internal/dbas/verifier"
	"reasonix/internal/provider"
	"reasonix/dbas/internal/dbas/plugin"
	"reasonix/dbas/internal/dbas/skill"
)

// banner is printed at demo startup.
const banner = `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   ██████╗ ██████╗  █████╗ ███████╗                          ║
║   ██╔══██╗██╔══██╗██╔══██╗██╔════╝                          ║
║   ██║  ██║██████╔╝███████║███████╗                          ║
║   ██║  ██║██╔══██╗██╔══██║╚════██║                          ║
║   ██████╔╝██████╔╝██║  ██║███████║                          ║
║   ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝                          ║
║                                                              ║
║   Disciplined Branching Agent System                         ║
║   Governance-first control plane for AI coding agents        ║
║                                                              ║
║   Version: 0.1.0-alpha                                       ║
║   Mode:    OFFLINE DEMO (no API keys needed)                 ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝

`

// RunDemo spawns 5 demo branches that walk through the full DBAS lifecycle.
// The branches show different governance scenarios in real-time.
func RunDemo(engine *branch.BranchEngine, store *persistence.BranchStore, dashboardPort int) {

	fmt.Print(banner)
	fmt.Printf("  Dashboard:  http://localhost:%d\n", dashboardPort)
	fmt.Printf("  API:        http://localhost:%d/api/dbas/state\n", dashboardPort)
	fmt.Printf("  Skills:     http://localhost:%d/api/dbas/skills\n", dashboardPort)
	fmt.Printf("  Plugins:    http://localhost:%d/api/dbas/plugins\n", dashboardPort)
	fmt.Printf("  Extensions: http://localhost:%d/api/dbas/extensions\n\n", dashboardPort)

	// Show what's available
	showSkills()
	showPlugins()

	baseDir, _ := os.Getwd()

	fmt.Println()
	slog.Info("=== DBAS OFFLINE DEMO ===",
		"dashboard", fmt.Sprintf("http://localhost:%d", dashboardPort))

	fmt.Println()
	fmt.Println("  Spawning 5 demo branches...")
	fmt.Println()

	// Branch 1: Happy Path — smooth execution
	go runHappyPath(engine, store, baseDir)
	time.Sleep(300 * time.Millisecond)

	// Branch 2: Retry Path — fails twice, then succeeds with escalation
	go runRetryPath(engine, store, baseDir)
	time.Sleep(300 * time.Millisecond)

	// Branch 3: Error Path — FSM enforcement catches illegal transition
	go runErrorPath(engine, store, baseDir)
	time.Sleep(300 * time.Millisecond)

	// Branch 4: Parallel task — demonstrates multibranch scheduling
	go runParallelTask(engine, store, baseDir)
	time.Sleep(300 * time.Millisecond)

	// Branch 5: Skill-guided task — uses frontend-react skills
	go runSkillTask(engine, store, baseDir)
}

func showSkills() {
	loader := skill.NewLoader("./skills", "./dbas/skills")
	skills, err := loader.LoadAll()
	if err != nil {
		slog.Warn("skill loader error", "err", err)
		return
	}
	slog.Info("DEMO: Skills loaded", "count", len(skills))
	for _, sk := range skills {
		slog.Debug("DEMO:   -> " + sk.Name, "version", sk.Version, "desc", sk.Description)
	}
}

func showPlugins() {
	r := plugin.Global()
	executors := r.ListExecutors()
	verifiers := r.ListVerifiers()
	slog.Info("DEMO: Plugin registry", "executors", len(executors), "verifiers", len(verifiers))
	for _, e := range executors {
		slog.Info("DEMO:   executor: " + e)
	}
	for _, v := range verifiers {
		slog.Info("DEMO:   verifier: " + v.Name, "cmd", v.Command)
	}
}

func mockProvider(name string, responses []string) provider.Provider {
	return &mockProv{name: name, responses: responses}
}

type mockProv struct {
	name      string
	responses []string
	callIndex int
}

func (m *mockProv) Name() string { return m.name }

func (m *mockProv) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	go func() {
		defer close(ch)
		if m.callIndex < len(m.responses) {
			ch <- provider.Chunk{Type: provider.ChunkText, Text: m.responses[m.callIndex]}
			m.callIndex++
		}
	}()
	return ch, nil
}

type mockRunner struct {
	failCount int
	callCount int
	workdir   string
}

func (m *mockRunner) Run(ctx context.Context, input string) error {
	m.callCount++

	if m.workdir != "" {
		os.MkdirAll(m.workdir, 0755)
		outputFile := filepath.Join(m.workdir, "output.txt")
		content := fmt.Sprintf("Demo execution #%d\nTask: %s\n", m.callCount, input)
		if len(input) > 60 {
			content = input[:60] + "...\n"
		}
		os.WriteFile(outputFile, []byte(content), 0644)
	}

	if m.callCount <= m.failCount {
		return fmt.Errorf("simulated execution failure #%d", m.callCount)
	}
	return nil
}

func runHappyPath(engine *branch.BranchEngine, store *persistence.BranchStore, baseDir string) {
	slot := experienceslot.DefaultSlot()
	deciderProv := mockProvider("demo-decider", []string{
		`{"task_id":"","target_state":"EXECUTING","execution_plan":"Write hello world program","tools_needed":["write_file"],"expected_output":"workspace/output.txt","confidence":0.95}"`,
	})
	reviewerProv := mockProvider("demo-reviewer", []string{
		`{"approved":true,"target_state":"EXECUTING","execution_input":"Write a hello world program to workspace/output.txt","review_notes":"Plan approved","modifications":[]}"`,
	})

	decider := brain.NewDecider(deciderProv)
	reviewer := brain.NewReviewer(reviewerProv)

	b := branch.NewBranch("demo-happy", nil, "demo-root-hash", "demo-branch-hash-happy", slot)
	store.InitBranch(b)

	workDir := filepath.Join(baseDir, "dbas-data", "branches", b.BranchID, "workspace")
	b.OutputPaths.WorkspaceDir = workDir

	mockExec := &mockRunner{workdir: workDir}
	verifierTeam := verifier.NewVerifier()

	reportDir := filepath.Dir(b.OutputPaths.ResultReport)
	os.MkdirAll(reportDir, 0755)
	os.WriteFile(b.OutputPaths.ResultReport, []byte(`{"status":"success"}`), 0644)

	coord := brain.NewDeciderCoordinator(decider, reviewer, mockExec, verifierTeam, nil, store, b)

	slog.Info("DEMO: [1/5] Happy Path starting — smooth execution flow")
	b.Logs = append(b.Logs, "[DEMO] Happy Path: task accepted, beginning workflow...")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	ctx := context.Background()
	if err := coord.Run(ctx, "Write a hello world program"); err != nil {
		slog.Warn("DEMO: Happy Path encountered expected minor issue", "err", err)
	}

	slog.Info("DEMO: [1/5] Happy Path complete", "final_state", b.CurrentState)
	b.Logs = append(b.Logs, "[DEMO] Happy Path complete")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)
}

func runRetryPath(engine *branch.BranchEngine, store *persistence.BranchStore, baseDir string) {
	slot := experienceslot.DefaultSlot()
	deciderProv := mockProvider("demo-retry-decider", []string{
		`{"task_id":"","target_state":"EXECUTING","execution_plan":"Refactor code with tests","tools_needed":["edit_file","bash"],"expected_output":"workspace/output.txt","confidence":0.85}"`,
	})
	reviewerProv := mockProvider("demo-retry-reviewer", []string{
		`{"approved":true,"target_state":"EXECUTING","execution_input":"Refactor the module and add tests","review_notes":"Plan approved with caution","modifications":[]}"`,
	})

	decider := brain.NewDecider(deciderProv)
	reviewer := brain.NewReviewer(reviewerProv)

	b := branch.NewBranch("demo-retry", nil, "demo-root-hash", "demo-branch-hash-retry", slot)
	store.InitBranch(b)

	workDir := filepath.Join(baseDir, "dbas-data", "branches", b.BranchID, "workspace")
	b.OutputPaths.WorkspaceDir = workDir

	mockExec := &mockRunner{failCount: 2, workdir: workDir}
	verifierTeam := verifier.NewVerifier()

	reportDir := filepath.Dir(b.OutputPaths.ResultReport)
	os.MkdirAll(reportDir, 0755)
	os.WriteFile(b.OutputPaths.ResultReport, []byte(`{"status":"success"}`), 0644)

	coord := brain.NewDeciderCoordinator(decider, reviewer, mockExec, verifierTeam, nil, store, b)

	slog.Info("DEMO: [2/5] Retry Path starting — will fail twice before succeeding")
	b.Logs = append(b.Logs, "[DEMO] Retry Path: starting task with known complexity...")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	ctx := context.Background()
	if err := coord.Run(ctx, "Refactor the module"); err != nil {
		slog.Warn("DEMO: Retry Path encountered expected issue", "err", err)
	}

	slog.Info("DEMO: [2/5] Retry Path complete", "final_state", b.CurrentState, "calls", mockExec.callCount)
	b.Logs = append(b.Logs, fmt.Sprintf("[DEMO] Retry Path complete after %d attempts", mockExec.callCount))
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)
}

func runErrorPath(engine *branch.BranchEngine, store *persistence.BranchStore, baseDir string) {
	slot := experienceslot.DefaultSlot()
	b := branch.NewBranch("demo-error", nil, "demo-root-hash", "demo-branch-hash-error", slot)
	store.InitBranch(b)

	b.Logs = append(b.Logs, "[DEMO] Error Path: attempting illegal state transition...")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(400 * time.Millisecond)
	b.Transition(fsm.StatePLAN, "Decider: proposed plan with invalid target")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(600 * time.Millisecond)
	b.Transition(fsm.StateEXECUTING, "Reviewer: approved execution")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(700 * time.Millisecond)
	b.Logs = append(b.Logs, "[DECIDER] Attempting to skip VERIFYING phase")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	slog.Info("DEMO: [3/5] Error Path — attempting illegal EXECUTING → ARCHIVE transition")
	err := b.Transition(fsm.StateARCHIVE, "Attempted illegal transition: EXECUTING → ARCHIVE")
	if err != nil {
		b.Logs = append(b.Logs, fmt.Sprintf("[FSM_ENFORCED] %v", err))
		store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
		store.SaveBranchMeta(b)
	}

	time.Sleep(500 * time.Millisecond)
	slog.Info("DEMO: [3/5] Error Path complete — FSM governance prevented illegal transition",
		"final_state", b.CurrentState)
	b.Logs = append(b.Logs, "[DEMO] Error Path complete — FSM governance enforced")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)
}

func runParallelTask(engine *branch.BranchEngine, store *persistence.BranchStore, baseDir string) {
	slot := experienceslot.DefaultSlot()
	b := branch.NewBranch("demo-parallel", nil, "demo-root-hash", "demo-branch-hash-parallel", slot)
	store.InitBranch(b)

	workDir := filepath.Join(baseDir, "dbas-data", "branches", b.BranchID, "workspace")
	b.OutputPaths.WorkspaceDir = workDir

	_ = workDir // used below for workspace

	b.Logs = append(b.Logs, "[DEMO] Parallel: simulating concurrent task execution...")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(500 * time.Millisecond)
	b.Transition(fsm.StatePLAN, "Decider: parallel execution plan")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(700 * time.Millisecond)
	b.Transition(fsm.StateEXECUTING, "Reviewer: approved parallel execution")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	// Run two mock sub-tasks to simulate parallelism
	go func() {
		subDir := filepath.Join(workDir, "subtask-a")
		os.MkdirAll(subDir, 0755)
		os.WriteFile(filepath.Join(subDir, "result.txt"), []byte("subtask-a: ok"), 0644)
		b.Logs = append(b.Logs, "[PARALLEL] Subtask A completed")
		store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	}()
	go func() {
		subDir := filepath.Join(workDir, "subtask-b")
		os.MkdirAll(subDir, 0755)
		os.WriteFile(filepath.Join(subDir, "result.txt"), []byte("subtask-b: ok"), 0644)
		b.Logs = append(b.Logs, "[PARALLEL] Subtask B completed")
		store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	}()

	time.Sleep(800 * time.Millisecond)
	b.Transition(fsm.StateVERIFYING, "Executor: parallel tasks complete")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	os.MkdirAll(filepath.Dir(b.OutputPaths.ResultReport), 0755)
	os.WriteFile(b.OutputPaths.ResultReport, []byte(`{"status":"success","subtasks":["subtask-a","subtask-b"]}`), 0644)

	time.Sleep(600 * time.Millisecond)
	b.Transition(fsm.StatePASS, "Verifier: all subtasks passed")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(400 * time.Millisecond)
	b.Transition(fsm.StateARCHIVE, "Cross-auditor: parallel execution verified")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	slog.Info("DEMO: [4/5] Parallel task complete", "final_state", b.CurrentState)
}

func runSkillTask(engine *branch.BranchEngine, store *persistence.BranchStore, baseDir string) {
	slot := experienceslot.DefaultSlot()
	b := branch.NewBranch("demo-skill-guided", nil, "demo-root-hash", "demo-branch-hash-skill", slot)
	store.InitBranch(b)

	workDir := filepath.Join(baseDir, "dbas-data", "branches", b.BranchID, "workspace")
	b.OutputPaths.WorkspaceDir = workDir

	b.Logs = append(b.Logs, "[DEMO] Skill Task: using frontend-react skill to guide React development...")
	b.Logs = append(b.Logs, "[DEMO] Skill-loaded instructions: React hooks, TypeScript patterns, CSS modules")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(400 * time.Millisecond)
	b.Transition(fsm.StatePLAN, "Decider: skill-guided frontend plan")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(600 * time.Millisecond)
	b.Transition(fsm.StateEXECUTING, "Reviewer: approved frontend plan with React hooks")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(800 * time.Millisecond)
	os.MkdirAll(workDir, 0755)
	os.WriteFile(filepath.Join(workDir, "App.tsx"), []byte(`import React, { useState } from "react";

export function App() {
  const [count, setCount] = useState(0);
  return <div onClick={() => setCount(c => c + 1)}>Count: {count}</div>;
}
`), 0644)

	b.Transition(fsm.StateVERIFYING, "Executor: React component created with TypeScript")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	os.MkdirAll(filepath.Dir(b.OutputPaths.ResultReport), 0755)
	os.WriteFile(b.OutputPaths.ResultReport, []byte(`{"status":"success","skill":"frontend-react","files":["App.tsx"]}`), 0644)

	time.Sleep(600 * time.Millisecond)
	b.Transition(fsm.StatePASS, "Verifier: component follows React hooks patterns from skill")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(400 * time.Millisecond)
	b.Transition(fsm.StateARCHIVE, "Cross-auditor: skill-guided development verified")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	slog.Info("DEMO: [5/5] Skill-guided task complete", "final_state", b.CurrentState, "skill", "frontend-react")
}
