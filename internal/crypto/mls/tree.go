package mls

import (
	"crypto/sha256"
	"fmt"
)

// InitializeTree creates a new MLS tree for a group.
func InitializeTree(members []string) *Tree {
	tree := &Tree{
		Nodes: make([]Node, 0),
		Size:  uint32(len(members)),
	}

	// Create leaf nodes for each member (indexed 0, 2, 4, ...)
	for i, member := range members {
		leafIndex := uint32(i * 2)
		leafNode := Node{
			IsLeaf:       true,
			Index:        leafIndex,
			EncryptKey:   deriveEncryptKey(member),
			SignatureKey: deriveSignatureKey(member),
		}
		tree.addNode(leafNode)
	}

	// Build parent nodes up the tree
	tree.buildParentNodes()
	return tree
}

// addNode appends a node to the tree.
func (t *Tree) addNode(node Node) {
	t.Nodes = append(t.Nodes, node)
}

// buildParentNodes constructs the parent node levels.
func (t *Tree) buildParentNodes() {
	currentLevel := 0
	for {
		leaves := t.getNodesAtLevel(currentLevel)
		if len(leaves) <= 1 {
			break
		}

		// Pair up nodes and create parents
		parents := make([]Node, 0)
		for i := 0; i < len(leaves); i += 2 {
			if i+1 < len(leaves) {
				parent := Node{
					IsLeaf:     false,
					Index:      t.nextParentIndex(),
					EncryptKey: hashNodePair(leaves[i].EncryptKey, leaves[i+1].EncryptKey),
				}
				parents = append(parents, parent)
				t.addNode(parent)
			} else {
				// Odd node out; it becomes a parent of itself
				parent := Node{
					IsLeaf:     false,
					Index:      t.nextParentIndex(),
					EncryptKey: hashNode(leaves[i].EncryptKey),
				}
				parents = append(parents, parent)
				t.addNode(parent)
			}
		}
		currentLevel++
	}
}

// getNodesAtLevel returns all leaf nodes (level 0).
func (t *Tree) getNodesAtLevel(level int) []Node {
	if level != 0 {
		return nil // Simplified: only return leaves
	}
	leaves := make([]Node, 0)
	for _, node := range t.Nodes {
		if node.IsLeaf {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// nextParentIndex generates the next parent node index.
func (t *Tree) nextParentIndex() uint32 {
	maxIndex := uint32(0)
	for _, node := range t.Nodes {
		if node.Index > maxIndex {
			maxIndex = node.Index
		}
	}
	return maxIndex + 1
}

// AddLeaf adds a new member to the tree (for Add proposals).
func (t *Tree) AddLeaf(member string) error {
	newLeafIndex := t.Size * 2
	newLeaf := Node{
		IsLeaf:       true,
		Index:        newLeafIndex,
		EncryptKey:   deriveEncryptKey(member),
		SignatureKey: deriveSignatureKey(member),
	}

	t.addNode(newLeaf)
	t.Size++

	// Rebuild parent nodes
	oldNodes := t.Nodes
	t.Nodes = []Node{newLeaf}
	for _, node := range oldNodes {
		if node.IsLeaf {
			t.addNode(node)
		}
	}
	t.buildParentNodes()
	return nil
}

// RemoveLeaf marks a leaf as blank (for Remove proposals).
func (t *Tree) RemoveLeaf(index uint32) error {
	if index >= uint32(len(t.Nodes)) {
		return fmt.Errorf("invalid leaf index: %d", index)
	}

	node := &t.Nodes[index]
	if !node.IsLeaf {
		return fmt.Errorf("node at index %d is not a leaf", index)
	}

	// Blank the node
	node.EncryptKey = nil
	node.SignatureKey = nil

	// Invalidate parent nodes
	t.invalidateAncestors(index)
	return nil
}

// invalidateAncestors marks all parent nodes as needing update.
func (t *Tree) invalidateAncestors(leafIndex uint32) {
	parentIndex := (leafIndex + 1) / 2
	for parentIndex > 0 && parentIndex < uint32(len(t.Nodes)) {
		node := &t.Nodes[parentIndex]
		node.EncryptKey = nil
		parentIndex = (parentIndex + 1) / 2
	}
}

// UpdateLeaf updates encryption key for a leaf (for Update proposals).
func (t *Tree) UpdateLeaf(index uint32, newEncryptKey []byte) error {
	if index >= uint32(len(t.Nodes)) {
		return fmt.Errorf("invalid leaf index: %d", index)
	}

	node := &t.Nodes[index]
	if !node.IsLeaf {
		return fmt.Errorf("node at index %d is not a leaf", index)
	}

	node.EncryptKey = newEncryptKey
	t.invalidateAncestors(index)
	return nil
}

// ComputeTreeHash returns the hash of the tree root.
func (t *Tree) ComputeTreeHash() ([]byte, error) {
	if len(t.Nodes) == 0 {
		return nil, fmt.Errorf("empty tree")
	}

	// Find root (highest index parent)
	var root *Node
	for i := len(t.Nodes) - 1; i >= 0; i-- {
		if !t.Nodes[i].IsLeaf {
			root = &t.Nodes[i]
			break
		}
	}

	if root == nil {
		// Single-member tree: return leaf hash
		return hashNode(t.Nodes[0].EncryptKey), nil
	}

	// Hash the root node
	return hashNode(root.EncryptKey), nil
}

// deriveEncryptKey derives an encryption key from a member address.
func deriveEncryptKey(member string) []byte {
	h := sha256.Sum256(append([]byte("encrypt:"), []byte(member)...))
	return h[:]
}

// deriveSignatureKey derives a signature key from a member address.
func deriveSignatureKey(member string) []byte {
	h := sha256.Sum256(append([]byte("signature:"), []byte(member)...))
	return h[:]
}

// hashNode returns the hash of a single node.
func hashNode(key []byte) []byte {
	h := sha256.Sum256(key)
	return h[:]
}

// hashNodePair returns the hash of two nodes.
func hashNodePair(left, right []byte) []byte {
	h := sha256.Sum256(append(left, right...))
	return h[:]
}

// ParentIndex returns the parent index of a node.
func ParentIndex(index uint32) uint32 {
	return (index + 1) / 2
}

// LeftChildIndex returns the left child index of a parent.
func LeftChildIndex(index uint32) uint32 {
	return index * 2
}

// RightChildIndex returns the right child index of a parent.
func RightChildIndex(index uint32) uint32 {
	return index*2 + 1
}
