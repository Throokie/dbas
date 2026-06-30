# MCP Server Example for DBAS

This directory contains a complete, self-contained MCP (Model Context Protocol) server
written in Go. It demonstrates how to extend DBAS with custom tools via the MCP protocol.

## Tools Provided

| Tool | Description |
|------|-------------|
| `count_words` | Count words, characters, and sentences in text |
| `extract_keywords` | Extract the most frequent keywords (English, basic TF) |
| `format_json` | Pretty-print a JSON string |
| `base64` | Encode or decode base64 strings |
| `diff_lines` | Compare two texts line-by-line |

## How to Use with DBAS

### 1. Build the server

```bash
cd examples/mcp-server
go build -o mcp-example.exe .
```

### 2. Configure in dbas.toml

```toml
[[plugins]]
name = "text-tools"
type = "stdio"
command = "./mcp-example.exe"
args = []
```

### 3. Start DBAS

```bash
./dbas.exe -config dbas.toml
```

DBAS will automatically load the MCP plugin, discover its tools, and register them
as executor extensions prefixed with `mcp-text-tools-`.

## Test Standalone

You can test the MCP server directly from the command line:

```bash
# Initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./mcp-example.exe

# List tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./mcp-example.exe

# Call count_words
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"count_words","arguments":{"text":"Hello world! This is a test."}}}' | ./mcp-example.exe

# Call format_json
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"format_json","arguments":{"json":"{\"name\":\"test\",\"values\":[1,2,3]}"}}}' | ./mcp-example.exe
```

## Protocol

This server implements MCP spec `2024-11-05` with JSON-RPC 2.0 over stdio
(one JSON object per line). It supports:

- `initialize` — Protocol version negotiation
- `notifications/initialized` — Client readiness signal
- `tools/list` — Discover available tools
- `tools/call` — Execute a tool
