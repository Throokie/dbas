package verifier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
)

func TestBuildAuditPromptIncludesPhysicalEvidence(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "execution.log")
	reportPath := filepath.Join(root, "report.json")
	if err := os.WriteFile(logPath, []byte("go test ./... PASS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, []byte(`{"status":"success","summary":"done"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	b := branch.NewBranch("branch-audit", nil, "root", "branch", experienceslot.DefaultSlot())
	b.OutputPaths.WorkspaceDir = workspace
	b.OutputPaths.LogFile = logPath
	b.OutputPaths.ResultReport = reportPath

	prompt := (&LLMVerifier{}).buildAuditPrompt(b)
	for _, want := range []string{"Physical Evidence", "package main", "go test ./... PASS", `"summary":"done"`, "main.go"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
