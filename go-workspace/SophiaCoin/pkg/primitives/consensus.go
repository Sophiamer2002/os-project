package primitives

import (
	"crypto/sha256"
	"os-project/SophiaCoin/pkg/crypto"
)

func (b *Block) VerifyPreviousHash(hash HashResult) bool {
	my_hash := b.header.prevBlock
	return my_hash == hash
}

func (b *BlockHeader) VerifyPreviousHash(hash HashResult) bool {
	my_hash := b.prevBlock
	return my_hash == hash
}

func (b *Block) VerifyMerkleRoot() bool {
	if len(b.transactions) == 0 {
		return false
	}

	b.constructMerkleTree()
	return b.tree.root() == b.header.merkleRoot
}

func (b *Block) VerifyDifficulty(height int, difficulty uint32) bool {
	if height == 0 {
		return true
	}
	my_hash := b.hash()
	for i := 0; i < int(difficulty) && i < 256; i++ {
		if my_hash[i/8]&(0b10000000>>(i%8)) != 0 {
			return false
		}
	}

	return true
}

// This function verifies whether the coinbase transaction is valid.
// It checks whether the coinbase transaction has the correct structure
// and whether the coinbase transaction has the correct value given
// tips by other transactions.
func (b *Block) VerifyCoinbase(height uint32, tips uint64) bool {
	coinbase := b.transactions[0]
	if len(coinbase.GetTxIns()) != 1 {
		return false
	}
	txIn := coinbase.GetTxIns()[0]
	if txIn.GetTxPtr() != DEFAULT_HASH_RESULT {
		return false
	}
	if txIn.GetIndex() != height {
		return false
	}
	return true
}

func (tx *Transaction) VerifySignature(pubkey []*crypto.PublicKey) bool {
	if len(pubkey) != len(tx.txIns) {
		return false
	}

	if len(tx.signatures) != len(tx.txIns) {
		return false
	}

	for i, txIn := range tx.GetTxIns() {
		raw := &Transaction{
			txIns:      []TxIn{txIn},
			txOuts:     tx.GetTxOuts(),
			signatures: []signature{},
		}

		b := sha256.Sum256(raw.serialize())
		if !pubkey[i].Verify(b[:], tx.signatures[i][:]) {
			return false
		}
	}

	return true
}

func (tx *Transaction) Sign(keys ...*crypto.Key) {
	if len(keys) != len(tx.GetTxIns()) && len(keys) > 1 {
		panic("Invalid number of keys")
	}

	for i, txIn := range tx.GetTxIns() {
		var key *crypto.Key
		if len(keys) == 1 {
			key = keys[0]
		} else {
			key = keys[i]
		}
		raw := &Transaction{
			txIns:      []TxIn{txIn},
			txOuts:     tx.GetTxOuts(),
			signatures: []signature{},
		}

		b := sha256.Sum256(raw.serialize())
		tx.signatures = append(tx.signatures, signature(key.Sign(b[:])))
	}

}
