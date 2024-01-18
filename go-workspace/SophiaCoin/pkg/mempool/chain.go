package mempool

import (
	"errors"
	"os-project/SophiaCoin/pkg/crypto"
	pri "os-project/SophiaCoin/pkg/primitives"
)

type Chain struct {
	difficulty uint32

	blocks []*pri.Block
	txs    map[pri.HashResult]*pri.Transaction
	utxos  map[pri.HashResult][]bool
}

func newChain(difficulty uint32) *Chain {
	return &Chain{
		difficulty: difficulty,

		blocks: []*pri.Block{pri.GetGenesisBlock()},
		txs:    map[pri.HashResult]*pri.Transaction{},
		utxos:  map[pri.HashResult][]bool{},
	}
}

func (chain *Chain) AppendBlock(block *pri.Block) error {
	if !chain.VerifyBlock(block, true) {
		return errors.New("mempool.Chain.AppendBlock: Invalid block")
	}

	chain.blocks = append(chain.blocks, block)
	for idx, tx := range block.GetTransactions() {
		chain.txs[pri.Hash(&tx)] = &tx
		chain.utxos[pri.Hash(&tx)] = make([]bool, len(tx.GetTxOuts()))
		for i := range chain.utxos[pri.Hash(&tx)] {
			chain.utxos[pri.Hash(&tx)][i] = true
		}

		if idx == 0 {
			continue // coinbase transaction, doesn't need to check
		}
		for _, txIn := range tx.GetTxIns() {
			if !chain.utxos[txIn.GetTxPtr()][txIn.GetIndex()] {
				panic("mempool.Chain.AppendBlock: Invalid transaction")
			}
			chain.utxos[txIn.GetTxPtr()][txIn.GetIndex()] = false
		}
	}

	return nil
}

// The function verify whether the new block can be
// appended to the chain.
func (chain *Chain) VerifyBlock(block *pri.Block, check_difficulty bool) bool {
	// Check hash pointer in block header
	if !block.VerifyPreviousHash(pri.Hash(chain.blocks[len(chain.blocks)-1])) {
		return false
	}

	// Verify merkle root
	if !block.VerifyMerkleRoot() {
		return false
	}

	// Check difficulty
	if check_difficulty && !block.VerifyDifficulty(len(chain.blocks), chain.difficulty) {
		return false
	}

	// Check non-coinbase transactions
	ok, total_tips := chain.VerifyTransactions(block.GetTransactions()[1:])
	if !ok {
		return false
	}

	if !block.VerifyCoinbase(uint32(len(chain.blocks)), total_tips) {
		return false
	}

	return true
}

// This function checks whether the transactions are valid in the next block,
// and returns a bool indicating whether the transactions are valid and an
// uint64 indicating the total tips of the transactions(if the transactions
// are valid).
func (chain *Chain) VerifyTransactions(txs []pri.Transaction) (bool, uint64) {
	// Check txIn outpoints, No double spending
	// TODO: Check whether there are two transactions
	//		 in the current block that spend the same
	//		 outpoint.
	for _, tx := range txs {
		txIns := tx.GetTxIns()
		for _, txIn := range txIns {
			if _, ok := chain.utxos[txIn.GetTxPtr()]; !ok {
				return false, 0
			}
			if len(chain.utxos[txIn.GetTxPtr()]) <= int(txIn.GetIndex()) {
				return false, 0
			}
			if !chain.utxos[txIn.GetTxPtr()][txIn.GetIndex()] {
				return false, 0
			}
		}

		_, ok := chain.txs[pri.Hash(&tx)]
		if ok {
			return false, 0
		}
	}

	// Verify transaction signatures
	for _, tx := range txs {
		pubKeys := make([]*crypto.PublicKey, 0, len(tx.GetTxIns()))
		for _, txIn := range tx.GetTxIns() {
			pubKeys = append(pubKeys, chain.txs[txIn.GetTxPtr()].GetTxOuts()[txIn.GetIndex()].GetPubKey())
		}

		if !tx.VerifySignature(pubKeys) {
			return false, 0
		}
	}

	// Check total outs equals total ins
	var total_tips uint64 = 0
	for _, tx := range txs {
		var tips int64 = 0
		txIns := tx.GetTxIns()
		for _, txIn := range txIns {
			tips += int64(chain.txs[txIn.GetTxPtr()].GetTxOuts()[txIn.GetIndex()].GetValue())
		}

		txOuts := tx.GetTxOuts()
		for _, txOuts := range txOuts {
			tips -= int64(txOuts.GetValue())
		}

		if tips < 0 {
			return false, 0
		}
		total_tips += uint64(tips)
	}

	return true, total_tips
}

func (chain *Chain) RollbackBlock() error {
	if len(chain.blocks) == 1 {
		panic("mempool.Chain.RollbackBlock: Cannot rollback genesis block")
	}
	block := chain.blocks[len(chain.blocks)-1]
	chain.blocks = chain.blocks[:len(chain.blocks)-1]
	for i, tx := range block.GetTransactions() {
		delete(chain.txs, pri.Hash(&tx))
		delete(chain.utxos, pri.Hash(&tx))

		if i == 0 {
			continue // coinbase transaction, doesn't need to deal with txins
		}
		for _, txIn := range tx.GetTxIns() {
			chain.utxos[txIn.GetTxPtr()][txIn.GetIndex()] = true
		}
	}

	return nil
}

func (chain *Chain) Copy() *Chain {
	newChain := newChain(chain.difficulty)
	newChain.blocks = make([]*pri.Block, 0, len(chain.blocks))
	newChain.blocks = append(newChain.blocks, chain.blocks...)

	for hash, tx := range chain.txs {
		newChain.txs[hash] = tx
	}

	for hash, utxo := range chain.utxos {
		copy(newChain.utxos[hash], utxo)
	}

	return newChain
}
