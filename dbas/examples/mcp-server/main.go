// MCP Server Example — A complete, self-contained MCP server for DBAS.
//
// This server implements the Model Context Protocol (MCP) over stdio JSON-RPC.
// It provides text-processing tools that can be loaded by DBAS as executor extensions.
//
// Build:
//   cd examples/mcp-server
//   go build -o mcp-example.exe .
//
// Test standalone:
//   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./mcp-example.exe
//
// Use with DBAS:
//   [[plugins]]
//   name = "text-tools"
//   type = "stdio"
//   command = "./mcp-example.exe"
//   args = []
//
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode"
)

// --- JSON-RPC types ---

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcErr     `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type callToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// --- Tool definitions ---

var tools = []map[string]interface{}{
	{
		"name":        "count_words",
		"description": "Count words, characters, and sentences in the provided text",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{"type": "string", "description": "Text to analyze"},
			},
			"required": []string{"text"},
		},
	},
	{
		"name":        "extract_keywords",
		"description": "Extract the most frequent keywords from text (English only, basic TF)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text":  map[string]interface{}{"type": "string", "description": "Text to analyze"},
				"limit": map[string]interface{}{"type": "number", "description": "Max keywords to return (default 10)"},
			},
			"required": []string{"text"},
		},
	},
	{
		"name":        "format_json",
		"description": "Pretty-print a JSON string with indentation",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json":  map[string]interface{}{"type": "string", "description": "Raw JSON string to format"},
				"indent": map[string]interface{}{"type": "number", "description": "Indentation spaces (default 2)"},
			},
			"required": []string{"json"},
		},
	},
	{
		"name":        "base64",
		"description": "Encode or decode base64 strings",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text":    map[string]interface{}{"type": "string", "description": "Text to encode or decode"},
				"action":  map[string]interface{}{"type": "string", "description": "encode or decode", "enum": []string{"encode", "decode"}},
			},
			"required": []string{"text", "action"},
		},
	},
	{
		"name":        "diff_lines",
		"description": "Compare two texts and show added, removed, and unchanged lines",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"old_text": map[string]interface{}{"type": "string", "description": "Original text"},
				"new_text": map[string]interface{}{"type": "string", "description": "New text"},
			},
			"required": []string{"old_text", "new_text"},
		},
	},
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
						"tools": map[string]interface{}{"listChanged": false},
					},
					"serverInfo": map[string]interface{}{
						"name":    "dbas-mcp-example",
						"version": "0.1.0",
					},
				},
			})

		case "notifications/initialized":
			// No response

		case "tools/list":
			enc.Encode(response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  map[string]interface{}{"tools": tools},
			})

		case "tools/call":
			var params callToolParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				enc.Encode(response{
					JSONRPC: "2.0", ID: req.ID,
					Error: &rpcErr{Code: -32602, Message: "Invalid params: " + err.Error()},
				})
				continue
			}

			result := handleToolCall(params.Name, params.Arguments)
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
				JSONRPC: "2.0", ID: req.ID,
				Error: &rpcErr{Code: -32601, Message: "Method not found: " + req.Method},
			})
		}
	}
}

func handleToolCall(name string, args map[string]interface{}) string {
	switch name {
	case "count_words":
		text, _ := args["text"].(string)
		return countWords(text)
	case "extract_keywords":
		text, _ := args["text"].(string)
		limit, _ := args["limit"].(float64)
		if limit <= 0 {
			limit = 10
		}
		return extractKeywords(text, int(limit))
	case "format_json":
		rawJSON, _ := args["json"].(string)
		indent, _ := args["indent"].(float64)
		if indent <= 0 {
			indent = 2
		}
		return formatJSON(rawJSON, int(indent))
	case "base64":
		text, _ := args["text"].(string)
		action, _ := args["action"].(string)
		return handleBase64(text, action)
	case "diff_lines":
		oldText, _ := args["old_text"].(string)
		newText, _ := args["new_text"].(string)
		return diffLines(oldText, newText)
	default:
		return fmt.Sprintf("Unknown tool: %s", name)
	}
}

func countWords(text string) string {
	words := strings.Fields(text)
	charCount := len([]rune(text))
	sentenceCount := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' {
			sentenceCount++
		}
	}

	return fmt.Sprintf(`Words: %d
Characters: %d
Sentences: %d
Avg word length: %.1f`,
		len(words),
		charCount,
		sentenceCount,
		avgWordLength(words))
}

func avgWordLength(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	total := 0
	for _, w := range words {
		total += len([]rune(w))
	}
	return math.Round(float64(total)/float64(len(words))*10) / 10
}

func extractKeywords(text string, limit int) string {
	// Simple TF-based keyword extraction (English only)
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "shall": true,
		"should": true, "may": true, "might": true, "can": true, "could": true,
		"it": true, "its": true, "this": true, "that": true, "these": true, "those": true,
		"i": true, "you": true, "he": true, "she": true, "we": true, "they": true,
		"me": true, "him": true, "her": true, "us": true, "them": true,
		"my": true, "your": true, "his": true, "our": true, "their": true,
		"not": true, "no": true, "nor": true, "so": true, "if": true, "then": true,
		"as": true, "from": true, "into": true, "over": true, "under": true, "up": true,
		"out": true, "about": true, "than": true, "also": true, "just": true, "very": true,
	}

	words := strings.Fields(strings.ToLower(text))
	freq := make(map[string]int)
	for _, w := range words {
		// Remove punctuation
		w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) })
		if w == "" || stopWords[w] {
			continue
		}
		freq[w]++
	}

	type kv struct{ k string; v int }
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	// Simple sort (bubble for simplicity in a small list)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].v > sorted[i].v {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if limit > len(sorted) {
		limit = len(sorted)
	}
	sorted = sorted[:limit]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Top %d keywords:\n\n", limit))
	for _, kv := range sorted {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", kv.k, kv.v))
	}
	return sb.String()
}

func formatJSON(rawJSON string, indent int) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return fmt.Sprintf("Error: invalid JSON — %s", err.Error())
	}
	indentStr := strings.Repeat(" ", indent)
	formatted, err := json.MarshalIndent(parsed, "", indentStr)
	if err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	return string(formatted)
}

func handleBase64(text, action string) string {
	switch action {
	case "encode":
		return encodeBase64([]byte(text))
	case "decode":
		decoded, err := decodeBase64(text)
		if err != nil {
			return fmt.Sprintf("Error: %s", err.Error())
		}
		return string(decoded)
	default:
		return fmt.Sprintf("Error: unknown base64 action %q (use encode or decode)", action)
	}
}

func encodeBase64(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	i := 0
	for i < len(data) {
		var b [3]byte
		n := 0
		for j := 0; j < 3 && i < len(data); j++ {
			b[j] = data[i]
			i++
			n++
		}
		if n == 1 {
			result.WriteByte(alphabet[b[0]>>2])
			result.WriteByte(alphabet[(b[0]&0x03)<<4])
			result.WriteString("==")
		} else if n == 2 {
			result.WriteByte(alphabet[b[0]>>2])
			result.WriteByte(alphabet[(b[0]&0x03)<<4|b[1]>>4])
			result.WriteByte(alphabet[(b[1]&0x0f)<<2])
			result.WriteByte('=')
		} else {
			result.WriteByte(alphabet[b[0]>>2])
			result.WriteByte(alphabet[(b[0]&0x03)<<4|b[1]>>4])
			result.WriteByte(alphabet[(b[1]&0x0f)<<2|b[2]>>6])
			result.WriteByte(alphabet[b[2]&0x3f])
		}
	}
	return result.String()
}

func decodeBase64(s string) ([]byte, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	decMap := make(map[byte]int)
	for i := 0; i < 64; i++ {
		decMap[alphabet[i]] = i
	}

	s = strings.TrimRight(s, "=")
	var result []byte
	for i := 0; i < len(s); i += 4 {
		var b [4]int
		for j := 0; j < 4; j++ {
			if i+j < len(s) {
				v, ok := decMap[s[i+j]]
				if !ok {
					return nil, fmt.Errorf("invalid base64 character %q", s[i+j])
				}
				b[j] = v
			}
		}
		result = append(result, byte(b[0]<<2|b[1]>>4))
		if i+2 < len(s) {
			result = append(result, byte(b[1]<<4|b[2]>>2))
		}
		if i+3 < len(s) {
			result = append(result, byte(b[2]<<6|b[3]))
		}
	}
	return result, nil
}

func diffLines(oldText, newText string) string {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	// Simple LCS-based diff (simplified for demonstration)
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Diff (%d old lines vs %d new lines):\n\n", len(oldLines), len(newLines)))

	added := 0
	removed := 0
	unchanged := 0

	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			sb.WriteString(fmt.Sprintf("  %s\n", oldLines[i]))
			i++
			j++
			unchanged++
		} else if j < len(newLines) && (i >= len(oldLines) || fuzzyMatch(newLines[j], oldLines, i) < 0) {
			sb.WriteString(fmt.Sprintf("+ %s\n", newLines[j]))
			j++
			added++
		} else if i < len(oldLines) {
			sb.WriteString(fmt.Sprintf("- %s\n", oldLines[i]))
			i++
			removed++
		} else {
			break
		}
	}

	sb.WriteString(fmt.Sprintf("\n---\n%d added, %d removed, %d unchanged", added, removed, unchanged))
	return sb.String()
}

func fuzzyMatch(line string, lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if lines[i] == line {
			return i
		}
	}
	return -1
}
