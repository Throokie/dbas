// Package demo provides an offline demo mode for DBAS.
// It uses mock providers and simulated executors to demonstrate
// the full FSM governance lifecycle without requiring any API keys.
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

// RunDemo spawns 3 demo branches that walk through the full DBAS lifecycle:
//   Branch A (Happy Path):  INIT 鈫?PLAN 鈫?EXECUTING 鈫?VERIFYING 鈫?PASS 鈫?ARCHIVE
//   Branch B (Retry Path):  ... 鈫?FAIL_RETREAT 鈫?EXECUTING 鈫?... 鈫?ARCHIVE
//   Branch C (Error Path):  ... 鈫?ERROR (FSM enforcement)
//
// Each branch uses mock providers (no API key required) and demonstrates
// the Decider 鈫?Reviewer 鈫?Executor 鈫?Verifier 鈫?Archive governance flow.
func RunDemo(engine *branch.BranchEngine, store *persistence.BranchStore, dashboardPort int) {

	// Display extensibility system info
	showSkills()
	showPlugins()

	baseDir, _ := os.Getwd()

	slog.Info("=== DBAS OFFLINE DEMO ===",
		"dashboard", fmt.Sprintf("http://localhost:%d", dashboardPort))

	// Happy Path
	go runHappyPath(engine, store, baseDir)
	time.Sleep(100 * time.Millisecond)

	// Retry Path
	go runRetryPath(engine, store, baseDir)
	time.Sleep(100 * time.Millisecond)

	// Error Path
	go runErrorPath(engine, store, baseDir)
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

	// Create a workspace directory and a simulated output file
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

	// Create the report file so verifier doesn`t fail
	reportDir := filepath.Dir(b.OutputPaths.ResultReport)
	os.MkdirAll(reportDir, 0755)
	os.WriteFile(b.OutputPaths.ResultReport, []byte(`{"status":"success"}`), 0644)

	coord := brain.NewDeciderCoordinator(decider, reviewer, mockExec, verifierTeam, nil, store, b)

	slog.Info("DEMO: Happy Path starting")
	b.Logs = append(b.Logs, "[DEMO] Happy Path: task accepted, beginning workflow...")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	ctx := context.Background()
	if err := coord.Run(ctx, "Write a hello world program"); err != nil {
		slog.Warn("DEMO: Happy Path encountered expected minor issue", "err", err)
	}

	slog.Info("DEMO: Happy Path complete", "final_state", b.CurrentState)
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

	// Executor fails TWICE before succeeding (demonstrates retry loop)
	mockExec := &mockRunner{failCount: 2, workdir: workDir}
	verifierTeam := verifier.NewVerifier()

	reportDir := filepath.Dir(b.OutputPaths.ResultReport)
	os.MkdirAll(reportDir, 0755)
	os.WriteFile(b.OutputPaths.ResultReport, []byte(`{"status":"success"}`), 0644)

	coord := brain.NewDeciderCoordinator(decider, reviewer, mockExec, verifierTeam, nil, store, b)

	slog.Info("DEMO: Retry Path starting (will fail twice before succeeding)")
	b.Logs = append(b.Logs, "[DEMO] Retry Path: starting task with known complexity...")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	ctx := context.Background()
	if err := coord.Run(ctx, "Refactor the module"); err != nil {
		slog.Warn("DEMO: Retry Path encountered expected issue", "err", err)
	}

	slog.Info("DEMO: Retry Path complete", "final_state", b.CurrentState, "calls", mockExec.callCount)
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

	// Walk through states normally, then attempt an illegal transition
	time.Sleep(400 * time.Millisecond)
	b.Transition(fsm.StatePLAN, "Decider: proposed plan with invalid target")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	time.Sleep(600 * time.Millisecond)
	b.Transition(fsm.StateEXECUTING, "Reviewer: approved execution")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	// Illegal transition: EXECUTING 鈫?ARCHIVE (skip VERIFYING)
	time.Sleep(700 * time.Millisecond)
	b.Logs = append(b.Logs, "[DECIDER] Attempting to skip VERIFYING phase")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)

	err := b.Transition(fsm.StateARCHIVE, "Attempted illegal transition: EXECUTING 鈫?ARCHIVE")
	if err != nil {
		b.Logs = append(b.Logs, fmt.Sprintf("[FSM_ENFORCED] %v", err))
		store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
		store.SaveBranchMeta(b)
	}

	time.Sleep(500 * time.Millisecond)
	slog.Info("DEMO: Error Path complete - FSM governance prevented illegal transition",
		"final_state", b.CurrentState)
	b.Logs = append(b.Logs, "[DEMO] Error Path complete - FSM governance enforced")
	store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
	store.SaveBranchMeta(b)
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
		slog.Info("DEMO:   -> " + sk.Name, "version", sk.Version, "desc", sk.Description)
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
