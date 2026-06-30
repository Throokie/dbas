// Package extend provides built-in extension implementations for DBAS.
package extend

import (
	"fmt"
	"os/exec"
	"runtime"

	"reasonix/dbas/internal/dbas/plugin"
)

func init() {
	RegisterAll()
}

func RegisterAll() {
	r := plugin.Global()
	r.RegisterExtension(&NativeExecutors{})
	r.RegisterExtension(&ToolVerifiers{})
	r.RegisterExtension(&ChannelNotifiers{})
}

// NativeExecutors provides built-in executor backends.
type NativeExecutors struct{}

func (e *NativeExecutors) Name() string        { return "native-executors" }
func (e *NativeExecutors) Description() string  { return "Built-in executor backends: reasonix-native, command, docker" }
func (e *NativeExecutors) Version() string      { return "1.0.0" }

func (e *NativeExecutors) RegisterExecutors(r plugin.ExecutorRegistry) {
	r.RegisterExecutor("reasonix-native", func(cfg map[string]interface{}) (interface{}, error) {
		return nil, fmt.Errorf("reasonix-native executor requires provider config")
	})
	r.RegisterExecutor("command", func(cfg map[string]interface{}) (interface{}, error) {
		return nil, fmt.Errorf("command executor requires command config")
	})
	r.RegisterExecutor("docker", func(cfg map[string]interface{}) (interface{}, error) {
		return newDockerExecutor(cfg)
	})
}
func (e *NativeExecutors) RegisterVerifiers(r plugin.VerifierRegistry) {}
func (e *NativeExecutors) RegisterNotifiers(r plugin.NotifierRegistry) {}

type dockerExecutor struct{}

func newDockerExecutor(cfg map[string]interface{}) (*dockerExecutor, error) {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker not available: %w", err)
	}
	return &dockerExecutor{}, nil
}

// ToolVerifiers provides built-in verification commands.
type ToolVerifiers struct{}

func (v *ToolVerifiers) Name() string       { return "tool-verifiers" }
func (v *ToolVerifiers) Description() string { return "Built-in verification commands: go-test, pytest, node-test" }
func (v *ToolVerifiers) Version() string     { return "1.0.0" }

func (v *ToolVerifiers) RegisterExecutors(r plugin.ExecutorRegistry)    {}
func (v *ToolVerifiers) RegisterNotifiers(r plugin.NotifierRegistry)     {}
func (v *ToolVerifiers) RegisterVerifiers(r plugin.VerifierRegistry) {
	r.RegisterVerifier(plugin.VerifierTool{Name: "go-test", Command: "go test ./... && go vet ./..."})
	r.RegisterVerifier(plugin.VerifierTool{Name: "pytest", Command: "python -m pytest && python -m flake8"})
	r.RegisterVerifier(plugin.VerifierTool{Name: "node-test", Command: "npm test && npm run lint"})
	if runtime.GOOS == "windows" {
		r.RegisterVerifier(plugin.VerifierTool{Name: "powershell-analyzer", Command: `powershell -Command "Get-ChildItem -Recurse *.ps1 | Select-String -Pattern 'TODO'"`})
	}
}

// ChannelNotifiers provides built-in notification channels.
type ChannelNotifiers struct{}

func (n *ChannelNotifiers) Name() string       { return "channel-notifiers" }
func (n *ChannelNotifiers) Description() string { return "Built-in notification channels: console, feishu" }
func (n *ChannelNotifiers) Version() string     { return "1.0.0" }

func (n *ChannelNotifiers) RegisterExecutors(r plugin.ExecutorRegistry)   {}
func (n *ChannelNotifiers) RegisterVerifiers(r plugin.VerifierRegistry)    {}
func (n *ChannelNotifiers) RegisterNotifiers(r plugin.NotifierRegistry) {
	r.RegisterNotifier("console", &consoleNotifier{})
}

type consoleNotifier struct{}

func (c *consoleNotifier) Name() string { return "console" }
func (c *consoleNotifier) Notify(title, message string) error {
		fmt.Printf("[DBAS] %s: %s\n", title, message)

	return nil
}
