package mcpplugin

import (
	"context"
	"fmt"
	"time"

	"reasonix/dbas/internal/dbas/plugin"
)

// BridgeAdapter adapts MCP plugins into DBAS's plugin system.
// It discovers MCP tools and registers them as DBAS executor extensions.
type BridgeAdapter struct {
	loader  *Loader
	timeout time.Duration
}

// NewBridge creates an adapter that bridges MCP plugins into DBAS.
func NewBridge(loader *Loader) *BridgeAdapter {
	return &BridgeAdapter{
		loader:  loader,
		timeout: 60 * time.Second,
	}
}

// RegisterAll discovers tools from all loaded MCP plugins and registers them
// as DBAS executors and tools in the given registry.
func (b *BridgeAdapter) RegisterAll(registry *plugin.Registry) error {
	for _, name := range b.loader.List() {
		if err := b.RegisterPlugin(name, registry); err != nil {
			return fmt.Errorf("register plugin %q: %w", name, err)
		}
	}
	return nil
}

// RegisterPlugin discovers tools from a single MCP plugin and registers them.
func (b *BridgeAdapter) RegisterPlugin(name string, registry *plugin.Registry) error {
	p, ok := b.loader.Get(name)
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	// Discover tools
	tools, err := p.Discover()
	if err != nil {
		return fmt.Errorf("discover tools: %w", err)
	}

	if len(tools) == 0 {
		return nil // No tools to register
	}

	// Register each tool as a DBAS extension
	ext := &mcpExtension{
		pluginName: name,
		plugin:     p,
		tools:      tools,
		timeout:    b.timeout,
	}

	registry.RegisterExtension(ext)

	return nil
}

// SetTimeout sets the execution timeout for MCP tool calls.
func (b *BridgeAdapter) SetTimeout(d time.Duration) {
	b.timeout = d
}

// mcpExtension adapts an MCP plugin's tools into a DBAS Extension.
type mcpExtension struct {
	pluginName string
	plugin     *Plugin
	tools      []Tool
	timeout    time.Duration
}

func (e *mcpExtension) Name() string        { return "mcp-" + e.pluginName }
func (e *mcpExtension) Description() string  { return fmt.Sprintf("MCP plugin: %s (%d tools)", e.pluginName, len(e.tools)) }
func (e *mcpExtension) Version() string {
	info := e.plugin.Info()
	if info.Version != "" {
		return info.Version
	}
	return "1.0.0"
}

func (e *mcpExtension) RegisterExecutors(r plugin.ExecutorRegistry) {
	for _, tool := range e.tools {
		tool := tool // capture
		name := fmt.Sprintf("mcp-%s-%s", e.pluginName, tool.Name)
		r.RegisterExecutor(name, func(cfg map[string]interface{}) (interface{}, error) {
			return &mcpToolExecutor{
				plugin:  e.plugin,
				tool:    tool,
				timeout: e.timeout,
			}, nil
		})
	}
}

func (e *mcpExtension) RegisterVerifiers(r plugin.VerifierRegistry) {}
func (e *mcpExtension) RegisterNotifiers(r plugin.NotifierRegistry) {}

// mcpToolExecutor wraps an MCP tool for execution.
type mcpToolExecutor struct {
	plugin  *Plugin
	tool    Tool
	timeout time.Duration
}

// Execute runs the MCP tool and returns the result text.
func (e *mcpToolExecutor) Execute(args map[string]interface{}) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	result, err := e.plugin.CallTool(ctx, e.tool.Name, args)
	if err != nil {
		return "", fmt.Errorf("mcp tool %q: %w", e.tool.Name, err)
	}

	if result.IsError {
		return "", fmt.Errorf("mcp tool %q returned error", e.tool.Name)
	}

	// Concatenate all text content
	var sb string
	for _, c := range result.Content {
		if c.Type == "text" {
			if sb != "" {
				sb += "\n"
			}
			sb += c.Text
		}
	}
	return sb, nil
}

// MCPTool returns the underlying MCP tool descriptor.
func (e *mcpToolExecutor) MCPTool() Tool { return e.tool }
