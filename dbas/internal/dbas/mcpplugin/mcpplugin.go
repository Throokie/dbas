// Package mcpplugin provides a self-contained MCP (Model Context Protocol) client for DBAS.
// It implements JSON-RPC 2.0 over stdio transport, allowing DBAS to load and call
// tools from any MCP-compatible server process.
//
// MCP Specification: https://spec.modelcontextprotocol.io
// Transport: JSON-RPC 2.0 over stdin/stdout (one JSON object per line)
//
// Usage:
//
//    p, _ := mcpplugin.LoadPlugin("my-tool", []string{"node", "/path/to/server.js"})
//    defer p.Stop()
//
//    tools, _ := p.Discover()
//    result, _ := p.CallTool(ctx, "my-tool", map[string]interface{}{"arg1": "val1"})
package mcpplugin
