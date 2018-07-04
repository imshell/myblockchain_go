package main

import "crypto/sha256"

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := &MerkleNode{}

	// 叶子节点
	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHash := sha256.Sum256(append(left.Data, right.Data...))
		mNode.Data = prevHash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return mNode
}

func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	// 单数则把最后一个节点复制一份
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, dataum := range data {
		nodes = append(nodes, *NewMerkleNode(nil, nil, dataum))
	}

	for i := 0; i < len(data)/2; i++ {
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, *node)
		}

		nodes = newLevel
	}

	mTree := &MerkleTree{&nodes[0]}

	return mTree
}
