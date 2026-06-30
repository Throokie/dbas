package verifier

// tools.go 定义外部验证工具的标准封装。
// 每个工具都是一个函数，接收 workspace 路径，返回输出和错误。

import (
	"fmt"
	"os/exec"
	"strings"
)

// ToolFunc 是验证工具的签名。
type ToolFunc func(workspace string) (output string, err error)

// ToolRegistry 是验证工具的注册表。
type ToolRegistry struct {
	tools map[string]ToolFunc
}

// NewToolRegistry 初始化工具注册表。
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]ToolFunc)}
	// 注册默认工具
	r.Register("go-test", GoTest)
	r.Register("go-vet", GoVet)
	r.Register("go-build", GoBuild)
	r.Register("npm-test", NpmTest)
	r.Register("npm-lint", NpmLint)
	r.Register("py-test", PyTest)
	r.Register("py-flake8", PyFlake8)
	return r
}

// Register 注册一个工具。
func (r *ToolRegistry) Register(name string, fn ToolFunc) {
	r.tools[name] = fn
}

// Get 获取工具函数。
func (r *ToolRegistry) Get(name string) (ToolFunc, bool) {
	fn, ok := r.tools[name]
	return fn, ok
}

// List 返回所有已注册工具名称。
func (r *ToolRegistry) List() []string {
	out := make([]string, 0, len(r.tools))
	for name := range r.tools {
		out = append(out, name)
	}
	return out
}

// --- 内置工具实现 ---

// GoTest 运行 go test ./...
func GoTest(workspace string) (string, error) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// GoVet 运行 go vet ./...
func GoVet(workspace string) (string, error) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// GoBuild 运行 go build ./...
func GoBuild(workspace string) (string, error) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// NpmTest 运行 npm test
func NpmTest(workspace string) (string, error) {
	cmd := exec.Command("npm", "test")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// NpmLint 运行 npm run lint
func NpmLint(workspace string) (string, error) {
	cmd := exec.Command("npm", "run", "lint")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// PyTest 运行 pytest
func PyTest(workspace string) (string, error) {
	cmd := exec.Command("python", "-m", "pytest")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// PyFlake8 运行 flake8
func PyFlake8(workspace string) (string, error) {
	cmd := exec.Command("python", "-m", "flake8")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunTool 运行指定工具并格式化输出。
func RunTool(name string, workspace string, registry *ToolRegistry) (string, error) {
	fn, ok := registry.Get(name)
	if !ok {
		return "", fmt.Errorf("tool %s not registered", name)
	}
	return fn(workspace)
}

// RunTools 运行多个工具，返回汇总结果。
func RunTools(names []string, workspace string, registry *ToolRegistry) (map[string]string, error) {
	results := make(map[string]string)
	var errs []string
	for _, name := range names {
		out, err := RunTool(name, workspace, registry)
		results[name] = out
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return results, fmt.Errorf("tool failures: %s", strings.Join(errs, "; "))
	}
	return results, nil
}
