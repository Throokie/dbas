package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/internal/provider"
)

// LLMVerifier 是独立的 LLM 交叉审计器。
// 它使用一个独立的 LLM 实例对执行者的产出进行交叉验证。
// 它不听执行者的口头汇报，只读 workspace 文件 + 报告。
type LLMVerifier struct {
	provider provider.Provider
}

// NewLLMVerifier 初始化 LLM 交叉审计器。
func NewLLMVerifier(provider provider.Provider) *LLMVerifier {
	return &LLMVerifier{provider: provider}
}

// CrossAudit 对分支执行独立 LLM 交叉审计。
// 它会读取 workspace 文件和 report，让 LLM 判断产出是否合格。
func (v *LLMVerifier) CrossAudit(ctx context.Context, b *branch.Branch) (*Verdict, error) {
	// 1. 构建审计 prompt
	prompt := v.buildAuditPrompt(b)

	req := provider.Request{
		Messages: []provider.Message{
			{Role: "system", Content: llmVerifierSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		MaxTokens:   4096,
	}

	ch, err := v.provider.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm verifier stream: %w", err)
	}

	var raw strings.Builder
	for chunk := range ch {
		switch chunk.Type {
		case provider.ChunkText:
			raw.WriteString(chunk.Text)
		case provider.ChunkError:
			return nil, fmt.Errorf("llm verifier error: %w", chunk.Err)
		}
	}

	// 2. 解析 LLM 输出的 JSON
	verdict, err := parseLLMVerdict(raw.String())
	if err != nil {
		return nil, fmt.Errorf("llm verifier parse: %w", err)
	}

	return verdict, nil
}

func (v *LLMVerifier) buildAuditPrompt(b *branch.Branch) string {
	evidence := buildAuditEvidence(b)
	return fmt.Sprintf(`
# Audit Request

You are the DBAS Independent Cross-Auditor. Your task is to verify the quality of another AI's execution output.

## Branch Information
- Branch ID: %s
- Executor Model: %s
- Task: %s
- Retries: %d

## Workspace Directory
%s

## Result Report Path
%s

## Physical Evidence
%s

## Instructions
1. You must NOT trust the executor's verbal claims. You only evaluate physical files and evidence.
2. Check if the code compiles, tests pass, and requirements are met.
3. Look for obvious bugs, missing edge cases, or incomplete implementations.
4. Output a strict JSON verdict.

## Output Format
{
  "passed": true|false,
  "reason": "string (detailed explanation)",
  "details": ["finding 1", "finding 2"]
}
`, b.BranchID, b.ExecutorInfo.Model, b.BranchID, b.Retries,
		b.OutputPaths.WorkspaceDir, b.OutputPaths.ResultReport, evidence)
}

func buildAuditEvidence(b *branch.Branch) string {
	var out strings.Builder
	appendFileEvidence(&out, "Result Report", b.OutputPaths.ResultReport, 64*1024)
	appendTailEvidence(&out, "Execution Log Tail", b.OutputPaths.LogFile, 32*1024)
	appendWorkspaceEvidence(&out, b.OutputPaths.WorkspaceDir)
	if out.Len() == 0 {
		return "No physical evidence files were readable."
	}
	return out.String()
}

func appendWorkspaceEvidence(out *strings.Builder, workspace string) {
	if workspace == "" {
		return
	}
	info, err := os.Stat(workspace)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(out, "\n### Workspace\nNot readable: %s\n", workspace)
		return
	}

	const maxFiles = 40
	const maxSnippets = 12
	files := make([]string, 0, maxFiles)
	_ = filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(files) >= maxFiles {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(workspace, path)
		files = append(files, rel)
		return nil
	})

	fmt.Fprintf(out, "\n### Workspace File List\n")
	if len(files) == 0 {
		out.WriteString("No files found.\n")
		return
	}
	for _, f := range files {
		fmt.Fprintf(out, "- %s\n", f)
	}

	snippets := 0
	for _, rel := range files {
		if snippets >= maxSnippets || !isTextEvidenceFile(rel) {
			continue
		}
		appendFileEvidence(out, "Workspace File: "+rel, filepath.Join(workspace, rel), 16*1024)
		snippets++
	}
}

func appendFileEvidence(out *strings.Builder, title, path string, maxBytes int) {
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(out, "\n### %s\nNot readable: %s (%v)\n", title, path, err)
		return
	}
	truncated := false
	if len(data) > maxBytes {
		data = data[:maxBytes]
		truncated = true
	}
	fmt.Fprintf(out, "\n### %s\nPath: %s\n```\n%s\n```\n", title, path, string(data))
	if truncated {
		out.WriteString("[truncated]\n")
	}
}

func appendTailEvidence(out *strings.Builder, title, path string, maxBytes int) {
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(out, "\n### %s\nNot readable: %s (%v)\n", title, path, err)
		return
	}
	truncated := false
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
		truncated = true
	}
	fmt.Fprintf(out, "\n### %s\nPath: %s\n```\n%s\n```\n", title, path, string(data))
	if truncated {
		out.WriteString("[tail truncated]\n")
	}
}

func isTextEvidenceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".json", ".toml", ".yaml", ".yml", ".md", ".txt", ".html", ".css", ".sh", ".ps1":
		return true
	default:
		return false
	}
}

func parseLLMVerdict(raw string) (*Verdict, error) {
	jsonStr := extractJSON(raw)
	var v Verdict
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func extractJSON(raw string) string {
	start := strings.Index(raw, "```json")
	if start >= 0 {
		end := strings.Index(raw[start:], "```")
		if end > 7 {
			return strings.TrimSpace(raw[start+7 : start+end])
		}
	}
	start = strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

const llmVerifierSystemPrompt = `You are the DBAS Independent Cross-Auditor. Your role is strictly limited to:

1. Read and analyze the executor's output files from the workspace.
2. Evaluate whether the output meets the stated requirements.
3. Check for bugs, missing functionality, or incorrect logic.
4. Output ONLY a valid JSON verdict object.

CORE CONSTRAINTS:
- You do NOT trust the executor's summary or claims.
- You verify PHYSICAL evidence (file contents, code, test outputs).
- You do NOT write code or fix issues.
- Your output MUST be a valid JSON object.

OUTPUT SCHEMA:
{
  "passed": true|false,
  "reason": "string",
  "details": ["finding 1", "finding 2"]
}
`
