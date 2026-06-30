package mcpplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"strings"
	"testing"
	"time"
)

// mockMCPServer is a minimal MCP server that runs as a subprocess.
// It reads JSON-RPC from stdin and writes responses to stdout.
// The code is embedded as a Go string and run via "go run" from a temp file.
const mockServerCode = `package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type request struct {
	JSONRPC string          ` + "`json:\"jsonrpc\"`" + `
	ID      int             ` + "`json:\"id\"`" + `
	Method  string          ` + "`json:\"method\"`" + `
	Params  json.RawMessage ` + "`json:\"params,omitempty\"`" + `
}

type response struct {
	JSONRPC string      ` + "`json:\"jsonrpc\"`" + `
	ID      int         ` + "`json:\"id\"`" + `
	Result  interface{} ` + "`json:\"result,omitempty\"`" + `
	Error   *rpcErr     ` + "`json:\"error,omitempty\"`" + `
}

type rpcErr struct {
	Code    int    ` + "`json:\"code\"`" + `
	Message string ` + "`json:\"message\"`" + `
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			enc.Encode(response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{
							"listChanged": false,
						},
					},
					"serverInfo": map[string]interface{}{
						"name":    "mock-mcp-server",
						"version": "1.0.0",
					},
				},
			})

		case "notifications/initialized":
			// No response expected

		case "tools/list":
			enc.Encode(response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "echo",
							"description": "Echoes back the input text",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"text": map[string]interface{}{
										"type": "string",
									},
								},
								"required": []string{"text"},
							},
						},
						{
							"name":        "add",
							"description": "Adds two numbers",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"a": map[string]interface{}{"type": "number"},
									"b": map[string]interface{}{"type": "number"},
								},
								"required": []string{"a", "b"},
							},
						},
					},
				},
			})

		case "tools/call":
			var params struct {
				Name      string                 ` + "`json:\"name\"`" + `
				Arguments map[string]interface{} ` + "`json:\"arguments\"`" + `
			}
			json.Unmarshal(req.Params, &params)

			switch params.Name {
			case "echo":
				text, _ := params.Arguments["text"].(string)
				enc.Encode(response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"content": []map[string]interface{}{
							{"type": "text", "text": "echo: " + text},
						},
					},
				})
			case "add":
				a, _ := params.Arguments["a"].(float64)
				b, _ := params.Arguments["b"].(float64)
				result := fmt.Sprintf("%.0f", a+b)
				enc.Encode(response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"content": []map[string]interface{}{
							{"type": "text", "text": result},
						},
					},
				})
			default:
				enc.Encode(response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &rpcErr{
						Code:    -32601,
						Message: fmt.Sprintf("Tool not found: %s", params.Name),
					},
				})
			}

		default:
			enc.Encode(response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &rpcErr{
					Code:    -32601,
					Message: fmt.Sprintf("Method not found: %s", req.Method),
				},
			})
		}
	}
}
`

func writeMockServer(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "mcp-mock-*.go")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(mockServerCode); err != nil {
		os.Remove(f.Name())
		t.Fatalf("write mock server: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoadPlugin(t *testing.T) {
	mockPath := writeMockServer(t)
	defer os.Remove(mockPath)

	p, err := LoadPlugin("test-mcp", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	defer p.Stop()

	// Check server info
	info := p.Info()
	if info.Name != "mock-mcp-server" {
		t.Errorf("expected server name 'mock-mcp-server', got %q", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", info.Version)
	}
}

func TestDiscoverTools(t *testing.T) {
	mockPath := writeMockServer(t)
	defer os.Remove(mockPath)

	p, err := LoadPlugin("test-mcp", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	defer p.Stop()

	tools, err := p.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	if tools[0].Name != "echo" {
		t.Errorf("expected tool[0].Name='echo', got %q", tools[0].Name)
	}
	if tools[1].Name != "add" {
		t.Errorf("expected tool[1].Name='add', got %q", tools[1].Name)
	}
}

func TestCallEchoTool(t *testing.T) {
	mockPath := writeMockServer(t)
	defer os.Remove(mockPath)

	p, err := LoadPlugin("test-mcp", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	defer p.Stop()

	// Discover must be called first
	if _, err := p.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := context.Background()
	result, err := p.CallTool(ctx, "echo", map[string]interface{}{"text": "hello world"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != "echo: hello world" {
		t.Errorf("expected 'echo: hello world', got %q", result.Content[0].Text)
	}
}

func TestCallAddTool(t *testing.T) {
	mockPath := writeMockServer(t)
	defer os.Remove(mockPath)

	p, err := LoadPlugin("test-mcp", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	defer p.Stop()

	if _, err := p.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := context.Background()
	result, err := p.CallTool(ctx, "add", map[string]interface{}{"a": 40, "b": 2})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if !strings.Contains(result.Content[0].Text, "42") {
		t.Errorf("expected result containing '42', got %q", result.Content[0].Text)
	}
}

func TestCallUnknownTool(t *testing.T) {
	mockPath := writeMockServer(t)
	defer os.Remove(mockPath)

	p, err := LoadPlugin("test-mcp", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	defer p.Stop()

	if _, err := p.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := context.Background()
	_, err = p.CallTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
}

func TestLoader(t *testing.T) {
	mockPath := writeMockServer(t)
	defer os.Remove(mockPath)

	loader := NewLoader()

	p1, err := loader.Load("plugin-a", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("Load plugin-a failed: %v", err)
	}
	defer p1.Stop()

	p2, err := loader.Load("plugin-b", []string{"go", "run", mockPath})
	if err != nil {
		t.Fatalf("Load plugin-b failed: %v", err)
	}
	defer p2.Stop()

	list := loader.List()
	if len(list) != 2 {
		t.Errorf("expected 2 plugins, got %d: %v", len(list), list)
	}

	loader.StopAll()
}

func TestJSONRPCBase(t *testing.T) {
	// Test JSON serialization types
	req := rpcRequest{JSONRPC: "2.0", ID: 1, Method: "test"}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var decoded rpcRequest
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if decoded.JSONRPC != "2.0" || decoded.ID != 1 || decoded.Method != "test" {
		t.Errorf("unexpected decoded values: %+v", decoded)
	}

	// Test Tool serialization
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"x": map[string]interface{}{"type": "string"},
			},
		},
	}
	b, _ = json.Marshal(tool)
	var decodedTool Tool
	if err := json.Unmarshal(b, &decodedTool); err != nil {
		t.Fatalf("unmarshal tool: %v", err)
	}
	if decodedTool.Name != "test-tool" {
		t.Errorf("expected name 'test-tool', got %q", decodedTool.Name)
	}
}

func TestPluginStopIdempotent(t *testing.T) {
	p := &Plugin{Name: "test", closed: true}
	// Should not panic
	p.Stop()
}

func init() {
	// Ensure tests don't take forever
	_ = time.Second
	_ = fmt.Sprintf
}

