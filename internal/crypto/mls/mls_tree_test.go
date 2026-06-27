package mls

import (
	"testing"
)

func TestInitializeTree(t *testing.T) {
	members := []string{"alice@example.com", "bob@example.com"}
	tree := InitializeTree(members)

	if tree.Size != 2 {
		t.Errorf("tree size: got %d, want 2", tree.Size)
	}

	if len(tree.Nodes) == 0 {
		t.Error("tree has no nodes")
	}

	// Check leaf nodes
	leafCount := 0
	for _, node := range tree.Nodes {
		if node.IsLeaf {
			leafCount++
		}
	}

	if leafCount != 2 {
		t.Errorf("leaf count: got %d, want 2", leafCount)
	}
}

func TestAddLeaf(t *testing.T) {
	members := []string{"alice@example.com"}
	tree := InitializeTree(members)

	initialSize := tree.Size
	tree.AddLeaf("bob@example.com")

	if tree.Size != initialSize+1 {
		t.Errorf("tree size after add: got %d, want %d", tree.Size, initialSize+1)
	}

	// Count leaf nodes - should be 2
	leafCount := 0
	for _, node := range tree.Nodes {
		if node.IsLeaf && len(node.EncryptKey) > 0 {
			leafCount++
		}
	}

	if leafCount < 2 {
		t.Errorf("leaf count: got %d, want at least 2", leafCount)
	}
}

func TestRemoveLeaf(t *testing.T) {
	members := []string{"alice@example.com", "bob@example.com"}
	tree := InitializeTree(members)

	// Find a leaf node
	var leafIndex uint32
	for i, node := range tree.Nodes {
		if node.IsLeaf {
			leafIndex = uint32(i)
			break
		}
	}

	err := tree.RemoveLeaf(leafIndex)
	if err != nil {
		t.Errorf("remove leaf: %v", err)
	}

	// Check that the node is blanked
	if len(tree.Nodes[leafIndex].EncryptKey) != 0 {
		t.Error("leaf not properly blanked after removal")
	}
}

func TestUpdateLeaf(t *testing.T) {
	members := []string{"alice@example.com"}
	tree := InitializeTree(members)

	newEncryptKey := []byte("new_encrypt_key")
	leafIndex := uint32(0)

	err := tree.UpdateLeaf(leafIndex, newEncryptKey)
	if err != nil {
		t.Errorf("update leaf: %v", err)
	}

	if string(tree.Nodes[leafIndex].EncryptKey) != "new_encrypt_key" {
		t.Error("leaf encryption key not updated")
	}
}

func TestComputeTreeHash(t *testing.T) {
	members := []string{"alice@example.com", "bob@example.com"}
	tree := InitializeTree(members)

	hash, err := tree.ComputeTreeHash()
	if err != nil {
		t.Errorf("compute tree hash: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("tree hash length: got %d, want 32", len(hash))
	}

	// Deterministic
	hash2, _ := tree.ComputeTreeHash()
	if string(hash) != string(hash2) {
		t.Error("tree hash not deterministic")
	}
}

func TestTreeIndexing(t *testing.T) {
	testCases := []struct {
		index    uint32
		parent   uint32
		left     uint32
		right    uint32
	}{
		{2, 1, 4, 5},      // Leaf node 0: parent = (2+1)/2 = 1, left = 2*2 = 4, right = 2*2+1 = 5
		{3, 2, 6, 7},      // Leaf node 1: parent = (3+1)/2 = 2, left = 2*3 = 6, right = 2*3+1 = 7
		{4, 2, 8, 9},      // Parent: parent = (4+1)/2 = 2, left = 2*4 = 8, right = 2*4+1 = 9
		{5, 3, 10, 11},    // Another node: parent = (5+1)/2 = 3, left = 2*5 = 10, right = 2*5+1 = 11
	}

	for _, tc := range testCases {
		parent := ParentIndex(tc.index)
		if parent != tc.parent {
			t.Errorf("ParentIndex(%d): got %d, want %d", tc.index, parent, tc.parent)
		}

		left := LeftChildIndex(tc.index)
		if left != tc.left {
			t.Errorf("LeftChildIndex(%d): got %d, want %d", tc.index, left, tc.left)
		}

		right := RightChildIndex(tc.index)
		if right != tc.right {
			t.Errorf("RightChildIndex(%d): got %d, want %d", tc.index, right, tc.right)
		}
	}
}
