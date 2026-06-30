package multibranch

import (
	"testing"
)

func TestNewRadixTree(t *testing.T) {
	tree := NewRadixTree()
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}
	if len(tree.nodes) != 0 {
		t.Errorf("expected empty tree, got %d nodes", len(tree.nodes))
	}
}

func TestCreateRoot(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system prompt", "tool schema")

	if root == nil {
		t.Fatal("expected non-nil root")
	}
	if root.ID != "root" {
		t.Errorf("expected root ID 'root', got %s", root.ID)
	}
	if root.ZoneType != ZonePinned {
		t.Errorf("expected ZonePinned, got %s", root.ZoneType)
	}
	if root.ParentID != nil {
		t.Error("expected nil parent ID for root")
	}
	if len(root.Children) != 0 {
		t.Errorf("expected 0 children for root, got %d", len(root.Children))
	}
	if root.Hash == "" {
		t.Error("expected non-empty hash for root")
	}

	// Verify root is stored
	stored, ok := tree.GetNode("root")
	if !ok {
		t.Fatal("root not found in tree")
	}
	if stored.Hash != root.Hash {
		t.Error("stored root hash differs")
	}
}

func TestCreateRootGeneratesUniqueHash(t *testing.T) {
	tree1 := NewRadixTree()
	tree2 := NewRadixTree()

	root1 := tree1.CreateRoot("prompt A", "schema A")
	root2 := tree2.CreateRoot("prompt B", "schema B")

	if root1.Hash == root2.Hash {
		t.Error("expected different hashes for different contents")
	}
}

func TestCreateChild(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system", "tools")

	child := tree.CreateChild(root.ID, "shared context", ZoneAppend)
	if child == nil {
		t.Fatal("expected non-nil child")
	}
	if child.ParentID == nil || *child.ParentID != root.ID {
		t.Errorf("expected parent ID %s, got %v", root.ID, child.ParentID)
	}
	if child.ZoneType != ZoneAppend {
		t.Errorf("expected ZoneAppend, got %s", child.ZoneType)
	}
	if child.Hash == "" {
		t.Error("expected non-empty hash")
	}

	// Verify root tracks child
	updatedRoot, _ := tree.GetNode(root.ID)
	if len(updatedRoot.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(updatedRoot.Children))
	}
	if updatedRoot.Children[0] != child.ID {
		t.Errorf("expected child ID %s, got %s", child.ID, updatedRoot.Children[0])
	}
}

func TestCreateChildWithNonexistentParent(t *testing.T) {
	tree := NewRadixTree()
	child := tree.CreateChild("nonexistent", "content", ZoneAppend)
	if child != nil {
		t.Error("expected nil for nonexistent parent")
	}
}

func TestGetPrefixChain(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system", "tools")
	child := tree.CreateChild(root.ID, "context", ZoneAppend)

	chain := tree.GetPrefixChain(child.ID)
	if len(chain) != 2 {
		t.Fatalf("expected 2 nodes in chain, got %d", len(chain))
	}
	if chain[0].ID != root.ID {
		t.Errorf("expected first node to be root, got %s", chain[0].ID)
	}
	if chain[1].ID != child.ID {
		t.Errorf("expected second node to be child, got %s", chain[1].ID)
	}
}

func TestGetPrefixChainForRoot(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system", "tools")

	chain := tree.GetPrefixChain(root.ID)
	if len(chain) != 1 {
		t.Fatalf("expected 1 node in chain for root, got %d", len(chain))
	}
	if chain[0].ID != root.ID {
		t.Errorf("expected root ID, got %s", chain[0].ID)
	}
}

func TestGetPrefixChainNonexistentNode(t *testing.T) {
	tree := NewRadixTree()
	chain := tree.GetPrefixChain("nonexistent")
	if len(chain) != 0 {
		t.Errorf("expected empty chain for nonexistent node, got %d", len(chain))
	}
}

func TestComposeMessages(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system prompt", "tool schema")
	child := tree.CreateChild(root.ID, "shared context 1", ZoneAppend)
	grandchild := tree.CreateChild(child.ID, "shared context 2", ZoneAppend)

	messages := tree.ComposeMessages(grandchild.ID, "task instruction")
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages (system + 2 context + task), got %d", len(messages))
	}

	// First message should be system role
	if messages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got %s", messages[0].Role)
	}
	if messages[0].Content != "system prompt\ntool schema" {
		t.Errorf("unexpected system content: %s", messages[0].Content)
	}

	// Middle messages should be user role (context)
	for i := 1; i <= 2; i++ {
		if messages[i].Role != "user" {
			t.Errorf("message %d: expected role 'user', got %s", i, messages[i].Role)
		}
	}

	// Last message should be task instruction
	last := len(messages) - 1
	if messages[last].Role != "user" {
		t.Errorf("last message: expected role 'user', got %s", messages[last].Role)
	}
	if messages[last].Content != "task instruction" {
		t.Errorf("unexpected task content: %s", messages[last].Content)
	}
}

func TestComposeMessagesWithEmptyTask(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system", "tools")

	messages := tree.ComposeMessages(root.ID, "")
	if len(messages) != 1 {
		t.Fatalf("expected 1 message (system only), got %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Errorf("expected role 'system', got %s", messages[0].Role)
	}
}

func TestZonePinnedAndAppendTypes(t *testing.T) {
	if ZonePinned != "pinned" {
		t.Errorf("expected ZonePinned='pinned', got '%s'", ZonePinned)
	}
	if ZoneAppend != "append" {
		t.Errorf("expected ZoneAppend='append', got '%s'", ZoneAppend)
	}
	if ZoneEphemeral != "ephemeral" {
		t.Errorf("expected ZoneEphemeral='ephemeral', got '%s'", ZoneEphemeral)
	}
}

func TestMultipleChildren(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system", "tools")

	child1 := tree.CreateChild(root.ID, "context 1", ZoneAppend)
	child2 := tree.CreateChild(root.ID, "context 2", ZoneAppend)

	if child1 == nil || child2 == nil {
		t.Fatal("expected non-nil children")
	}
	if child1.ID == child2.ID {
		t.Error("expected different IDs for different children")
	}

	// Verify both children are tracked
	updatedRoot, _ := tree.GetNode(root.ID)
	if len(updatedRoot.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(updatedRoot.Children))
	}
}

func TestHashChainIntegrity(t *testing.T) {
	tree := NewRadixTree()
	root := tree.CreateRoot("system", "tools")
	child := tree.CreateChild(root.ID, "context", ZoneAppend)
	grandchild := tree.CreateChild(child.ID, "more context", ZoneEphemeral)

	// Root hash feeds into child hash
	if grandchild.Hash == child.Hash {
		t.Error("expected different hashes for different content")
	}
	if child.Hash == root.Hash {
		t.Error("expected different hashes for parent vs child")
	}

	// Same content should produce same hash in a new tree
	tree2 := NewRadixTree()
	root2 := tree2.CreateRoot("system", "tools")
	child2 := tree2.CreateChild(root2.ID, "context", ZoneAppend)
	grandchild2 := tree2.CreateChild(child2.ID, "more context", ZoneEphemeral)

	if grandchild.Hash != grandchild2.Hash {
		t.Error("expected same hash for identical content chains")
	}
}

func TestGetNodeNonexistent(t *testing.T) {
	tree := NewRadixTree()
	_, ok := tree.GetNode("nonexistent")
	if ok {
		t.Error("expected false for nonexistent node")
	}
}
