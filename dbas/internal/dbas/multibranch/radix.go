package multibranch

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// RadixNode 是 Radix-Tree 中的一个节点，代表一个上下文检查点。
// 根节点 = Zone 1（System Prompt + Tool Schema）
// 子节点 = Zone 2（Shared Context）
// 叶子节点 = Zone 3（Task Instruction）
type RadixNode struct {
	ID       string   `json:"id"`
	ParentID *string  `json:"parent_id,omitempty"`
	ZoneType ZoneType `json:"zone_type"`
	Content  string   `json:"content"`
	Hash     string   `json:"hash"`
	Children []string `json:"children"`
	CreatedAt time.Time `json:"created_at"`
}

// ZoneType 区分三区缓存。
type ZoneType string

const (
	ZonePinned   ZoneType = "pinned"   // Zone 1: System Prompt + Tool Schema（绝对静态）
	ZoneAppend   ZoneType = "append"   // Zone 2: Shared Context（追加只读）
	ZoneEphemeral ZoneType = "ephemeral" // Zone 3: Task Instruction（临时变动）
)

// RadixTree 是 DBAS 的上下文共享树。
// 所有分支共享同一棵树的前缀节点，只有叶子节点（Zone 3）独立。
type RadixTree struct {
	nodes map[string]*RadixNode
}

// NewRadixTree 初始化一棵新树。
func NewRadixTree() *RadixTree {
	return &RadixTree{nodes: make(map[string]*RadixNode)}
}

// CreateRoot 创建根节点（Zone 1）。
func (t *RadixTree) CreateRoot(systemPrompt, sortedToolSchema string) *RadixNode {
	h := sha256.New()
	h.Write([]byte(systemPrompt))
	h.Write([]byte(sortedToolSchema))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	root := &RadixNode{
		ID:       "root",
		ParentID: nil,
		ZoneType: ZonePinned,
		Content:  systemPrompt + "\n" + sortedToolSchema,
		Hash:     hash,
		Children: []string{},
		CreatedAt: time.Now().UTC(),
	}
	t.nodes[root.ID] = root
	return root
}

// CreateChild 创建子节点（Zone 2），继承父节点的前缀。
func (t *RadixTree) CreateChild(parentID, content string, zoneType ZoneType) *RadixNode {
	parent, ok := t.nodes[parentID]
	if !ok {
		return nil
	}

	h := sha256.New()
	h.Write([]byte(parent.Hash))
	h.Write([]byte(content))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	id := fmt.Sprintf("node-%s-%d", parentID, len(parent.Children))
	node := &RadixNode{
		ID:        id,
		ParentID:  &parentID,
		ZoneType:  zoneType,
		Content:   content,
		Hash:      hash,
		Children:  []string{},
		CreatedAt: time.Now().UTC(),
	}
	t.nodes[id] = node
	parent.Children = append(parent.Children, id)
	return node
}

// GetNode 查询节点。
func (t *RadixTree) GetNode(id string) (*RadixNode, bool) {
	node, ok := t.nodes[id]
	return node, ok
}

// GetPrefixChain 返回从根到该节点的完整前缀链。
// 用于构造 DeepSeek API 的 messages 数组，确保前缀缓存命中。
func (t *RadixTree) GetPrefixChain(nodeID string) []RadixNode {
	var chain []RadixNode
	current := nodeID
	for current != "" {
		node, ok := t.nodes[current]
		if !ok {
			break
		}
		chain = append([]RadixNode{*node}, chain...)
		if node.ParentID == nil {
			break
		}
		current = *node.ParentID
	}
	return chain
}

// ComposeMessages 将前缀链转换为 LLM messages 数组。
// Zone 1 内容作为 system 消息，Zone 2 作为 context 消息，Zone 3 作为 user 消息。
func (t *RadixTree) ComposeMessages(nodeID string, taskInstruction string) []Message {
	chain := t.GetPrefixChain(nodeID)
	var messages []Message

	for _, node := range chain {
		switch node.ZoneType {
		case ZonePinned:
			messages = append(messages, Message{Role: "system", Content: node.Content})
		case ZoneAppend:
			messages = append(messages, Message{Role: "user", Content: node.Content})
		}
	}

	// Zone 3: 任务指令作为最后一个 user 消息（Tail Append）
	if taskInstruction != "" {
		messages = append(messages, Message{Role: "user", Content: taskInstruction})
	}

	return messages
}

// Message 对应 LLM API 消息结构。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
