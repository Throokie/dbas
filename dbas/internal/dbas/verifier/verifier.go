package verifier

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"reasonix/dbas/internal/dbas/branch"
)

// Verifier 是 DBAS 的反欺骗验收队。
// Phase 1: 使用外部命令（go test, go vet, 等）进行物理验证。
// Phase 2: 增加独立 LLM 实例进行交叉审计。
type Verifier struct {
	// tools 是配置的外部检查工具映射
	// key: 语言/类型, value: 命令模板
	tools map[string]string
}

// Verdict 是验收结果。
type Verdict struct {
	Passed  bool     `json:"passed"`
	Reason  string   `json:"reason"`
	Details []string `json:"details"`
}

// NewVerifier 初始化验收队。
func NewVerifier() *Verifier {
	return &Verifier{
		tools: map[string]string{
			"go":        "go test ./... && go vet ./...",
			"python":    "python -m pytest && python -m flake8",
			"node":      "npm test && npm run lint",
			"ast-check": "echo 'AST check placeholder'", // Phase 1 占位
		},
	}
}

// Verify 对分支的工作成果进行验收。
func (v *Verifier) Verify(b *branch.Branch) (*Verdict, error) {
	workspace := b.OutputPaths.WorkspaceDir

	// 1. 检查 workspace 是否存在
	if _, err := os.Stat(workspace); os.IsNotExist(err) {
		return &Verdict{
			Passed:  false,
			Reason:  fmt.Sprintf("Workspace not found: %s", workspace),
			Details: []string{"No output files to verify"},
		}, nil
	}

	// 2. 检查 result report 是否存在
	reportPath := b.OutputPaths.ResultReport
	reportExists := false
	if _, err := os.Stat(reportPath); err == nil {
		reportExists = true
	}

	// 3. 根据分支使用的语言/工具选择验证命令
	toolName := b.Slot().Rules.VerifierRules.CodeSyntax
	cmdTemplate, ok := v.tools[toolName]
	if !ok {
		cmdTemplate = v.tools["ast-check"] // fallback
	}

	// 4. 执行验证命令
	details := []string{}
	passed := true
	reason := "All checks passed"

	if cmdTemplate != "echo 'AST check placeholder'" {
		cmd := shellCommand(cmdTemplate)
		cmd.Dir = workspace
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				details = append(details, trimmed)
			}
		}
		if err != nil {
			passed = false
			reason = fmt.Sprintf("Verification failed: %v", err)
		}
	} else {
		// Phase 1 占位：检查文件存在性
		details = append(details, "Phase 1: AST check placeholder")
		details = append(details, fmt.Sprintf("Workspace: %s", workspace))
		details = append(details, fmt.Sprintf("Report exists: %v", reportExists))
		files, _ := listFiles(workspace)
		for _, f := range files {
			details = append(details, fmt.Sprintf("  file: %s", f))
		}
	}

	// 5. 检查测试通过率（如果配置了）
	requiredRate := b.Slot().Rules.VerifierRules.TestPassRate
	if requiredRate == 100 && !passed {
		reason = fmt.Sprintf("Required test pass rate %d%% not met", requiredRate)
	}

	return &Verdict{
		Passed:  passed,
		Reason:  reason,
		Details: details,
	}, nil
}

func shellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}
	return exec.Command("sh", "-c", command)
}

// listFiles 递归列出目录下的文件（用于诊断）。
func listFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}

// RegisterTool 注册一个新的验证工具命令。
func (v *Verifier) RegisterTool(name, command string) {
	v.tools[name] = command
}
