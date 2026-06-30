package mcpplugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// JSON-RPC 2.0 message IDs are auto-incremented per plugin instance.
// MCP protocol version.
const mcpVersion = "2024-11-05"

// --- JSON-RPC types ---

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// --- MCP types ---

// InitializeResult is the response from an initialize call.
type InitializeResult struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo        `json:"serverInfo"`
}

// ServerCapabilities describes what the MCP server supports.
type ServerCapabilities struct {
	Tools *ToolCapability `json:"tools,omitempty"`
	// Resources, Prompts, etc. can be added as needed
}

// ToolCapability describes tool support.
type ToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo describes the MCP server itself.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool is a tool provided by an MCP server.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// CallToolResult is the result of calling a tool.
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent is the content of a tool result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// Other content types (image, resource) can be added
}

// Plugin represents a running MCP plugin process.
type Plugin struct {
	Name       string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Scanner
	stderr     *bufio.Scanner
	mu         sync.Mutex
	nextID     int
	info       ServerInfo
	caps       ServerCapabilities
	discovered bool
	tools      []Tool
	closed     bool
}

// LoadPlugin starts an MCP server process and performs initialization handshake.
// The args should be the command and arguments to start the MCP server.
// Example: LoadPlugin("my-tool", []string{"node", "/path/to/server.js"})
func LoadPlugin(name string, args []string) (*Plugin, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("mcpplugin: no command specified for plugin %q", name)
	}

	cmd := exec.Command(args[0], args[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpplugin %q: stdin pipe: %w", name, err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("mcpplugin %q: stdout pipe: %w", name, err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("mcpplugin %q: stderr pipe: %w", name, err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("mcpplugin %q: start: %w", name, err)
	}

	p := &Plugin{
		Name:   name,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(bufio.NewReaderSize(stdout, 256*1024)),
		stderr: bufio.NewScanner(bufio.NewReaderSize(stderr, 256*1024)),
		nextID: 1,
	}

	// Buffer stderr for log collection
	go func() {
		for p.stderr.Scan() {
			line := strings.TrimSpace(p.stderr.Text())
			if line != "" {
				// Stderr from MCP servers is typically logging/debug info
				// We'll just ignore it in the common case; callers can override
			}
		}
	}()

	// Perform initialization handshake
	if err := p.initialize(); err != nil {
		p.Stop()
		return nil, fmt.Errorf("mcpplugin %q: initialize: %w", name, err)
	}

	return p, nil
}

// initialize performs the MCP initialization handshake.
func (p *Plugin) initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := p.call(ctx, "initialize", map[string]interface{}{
		"protocolVersion": mcpVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "dbas",
			"version": "0.1.0",
		},
	})
	if err != nil {
		return fmt.Errorf("initialize call: %w", err)
	}

	var initResult InitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("parse initialize result: %w", err)
	}

	p.info = initResult.ServerInfo
	p.caps = initResult.Capabilities

	// Send initialized notification (no response expected)
	p.notify("notifications/initialized", map[string]interface{}{})

	return nil
}

// Discover discovers and caches tools from the MCP server.
// Returns cached tools on subsequent calls unless forceRefresh is true.
func (p *Plugin) Discover(forceRefresh ...bool) ([]Tool, error) {
	p.mu.Lock()
	if p.discovered && (len(forceRefresh) == 0 || !forceRefresh[0]) {
		tools := make([]Tool, len(p.tools))
		copy(tools, p.tools)
		p.mu.Unlock()
		return tools, nil
	}
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := p.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	var listResult struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("parse tools/list result: %w", err)
	}

	p.mu.Lock()
	p.tools = make([]Tool, len(listResult.Tools))
	copy(p.tools, listResult.Tools)
	p.discovered = true
	p.mu.Unlock()

	return listResult.Tools, nil
}

// CallTool calls a tool on the MCP server with the given arguments.
func (p *Plugin) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	result, err := p.call(ctx, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return nil, fmt.Errorf("tools/call %q: %w", name, err)
	}

	var callResult CallToolResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("parse tools/call result: %w", err)
	}

	return &callResult, nil
}

// Info returns the server info from initialization.
func (p *Plugin) Info() ServerInfo { return p.info }

// Capabilities returns the server capabilities from initialization.
func (p *Plugin) Capabilities() ServerCapabilities { return p.caps }

// Stop terminates the MCP server process gracefully.
func (p *Plugin) Stop() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	p.stdin.Close()
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}
}

// call sends a JSON-RPC request and waits for the response.
func (p *Plugin) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	p.mu.Lock()
	id := p.nextID
	p.nextID++
	p.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("plugin %q is closed", p.Name)
	}
	if _, err := p.stdin.Write(append(reqJSON, '\n')); err != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}
	p.mu.Unlock()

	// Read response (with context cancellation)
	respCh := make(chan *rpcResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		p.mu.Lock()
		hasMore := p.stdout.Scan()
		if !hasMore {
			err := p.stdout.Err()
			if err == nil {
				err = io.EOF
			}
			p.mu.Unlock()
			errCh <- fmt.Errorf("read response: %w", err)
			return
		}
		line := p.stdout.Text()
		p.mu.Unlock()

		var resp rpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			errCh <- fmt.Errorf("parse response: %w", err)
			return
		}
		respCh <- &resp
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// notify sends a JSON-RPC notification (no response expected).
func (p *Plugin) notify(method string, params interface{}) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	reqJSON, _ := json.Marshal(req)
	p.mu.Lock()
	p.stdin.Write(append(reqJSON, '\n'))
	p.mu.Unlock()
}

// Loader manages multiple MCP plugins.
type Loader struct {
	mu      sync.Mutex
	plugins map[string]*Plugin
}

// NewLoader creates an empty MCP plugin loader.
func NewLoader() *Loader {
	return &Loader{
		plugins: make(map[string]*Plugin),
	}
}

// Load starts and registers an MCP plugin.
func (l *Loader) Load(name string, args []string) (*Plugin, error) {
	p, err := LoadPlugin(name, args)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.plugins[name] = p
	l.mu.Unlock()

	return p, nil
}

// Get returns a loaded plugin by name.
func (l *Loader) Get(name string) (*Plugin, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.plugins[name]
	return p, ok
}

// List returns all loaded plugin names.
func (l *Loader) List() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	names := make([]string, 0, len(l.plugins))
	for name := range l.plugins {
		names = append(names, name)
	}
	return names
}

// StopAll terminates all loaded plugins.
func (l *Loader) StopAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, p := range l.plugins {
		p.Stop()
	}
	l.plugins = make(map[string]*Plugin)
}
