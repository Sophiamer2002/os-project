package primitives

import (
	"crypto/sha256"
)

// var DEFAULT_HASH_RESULT = HashResult(sha256.Sum256([]byte("")))

type merkleTree struct {
	nodes  [][]HashResult
	depth  uint32
	length uint32
}

func newMerkleTree(hash []HashResult) *merkleTree {
	n := len(hash)
	tree := merkleTree{
		nodes:  make([][]HashResult, 0),
		depth:  uint32(0),
		length: uint32(n),
	}
	if n == 0 {
		return &tree
	}

	tree.nodes = append(tree.nodes, make([]HashResult, n))
	for i := 0; i < n; i++ {
		tree.nodes[0][i] = hash[i]
	}
	tree.depth = 1
	for n > 1 {
		tree.nodes = append(tree.nodes, make([]HashResult, (n+1)/2))
		for i := 0; i < n; i += 2 {
			data := tree.nodes[tree.depth-1][i][:]
			if i+1 < n {
				data = append(data, tree.nodes[tree.depth-1][i+1][:]...)
				tree.nodes[tree.depth][i/2] = sha256.Sum256(data)
			} else {
				data = append(data, DEFAULT_HASH_RESULT[:]...)
				tree.nodes[tree.depth][i/2] = HashResult(data)
			}
		}
		tree.depth++
		n = (n + 1) / 2
	}

	return &tree
}

func (tree *merkleTree) root() HashResult {
	if tree.depth == 0 {
		return DEFAULT_HASH_RESULT
	}
	return tree.nodes[tree.depth-1][0]
}

func (tree *merkleTree) append(hash ...HashResult) {
	n := len(hash)
	if n == 0 {
		return
	}
	// Compute new depth and initialize new nodes
	tree.length += uint32(n)
	temp_length := tree.length
	new_depth := 0
	for temp_length > 0 {
		temp_length /= 2
		new_depth++
	}
	var_length := n
	for i := 0; i < new_depth; i++ {
		if i < int(tree.depth) {
			tree.nodes[i] = append(tree.nodes[i], make([]HashResult, var_length)...)
			var_length = (var_length + 1) / 2
		} else {
			tree.nodes = append(tree.nodes, make([]HashResult, var_length))
			var_length = (var_length + 1) / 2
		}
	}
	tree.depth = uint32(new_depth)

	// Append new nodes
	for i := 0; i < n; i++ {
		idx := tree.length - uint32(n) + uint32(i)
		tree.nodes[0][idx] = hash[i]
		temp_depth := 1
		for idx > 0 {
			if idx%2 == 1 {
				tree.nodes[temp_depth][idx/2] = sha256.Sum256(append(tree.nodes[tree.depth-1][idx-1][:], tree.nodes[tree.depth-1][idx][:]...))
			} else {
				tree.nodes[temp_depth][idx/2] = sha256.Sum256(append(tree.nodes[tree.depth-1][idx][:], DEFAULT_HASH_RESULT[:]...))
			}
			idx /= 2
			temp_depth++
		}
	}

	return
}

// We should use default hash result to represent empty hash in proof
func VerifyProof(root HashResult, proof []HashResult, leaf HashResult, idx int) bool {
	return true
	if len(proof) == 0 {
		return root == leaf && idx == 0
	}
	if idx == 0 {
		return root == leaf
	}
	for idx > 0 {
		if len(proof) == 0 {
			return false
		}
		if idx%2 == 1 {
			leaf = sha256.Sum256(append(proof[0][:], leaf[:]...))
		} else {
			leaf = sha256.Sum256(append(leaf[:], proof[0][:]...))
		}
		idx /= 2
		proof = proof[1:]
	}
	if len(proof) != 0 {
		return false
	}
	return root == leaf
}
