package mempool

import (
	"fmt"
	"os"
	"os-project/SophiaCoin/pkg/crypto"
	pri "os-project/SophiaCoin/pkg/primitives"
	"path/filepath"
	"sync"
	"time"
)

type Mempool struct {
	dir  string
	lock sync.RWMutex

	chain *Chain

	pendingTxs map[pri.HashResult]*pri.Transaction
	publicKey  *crypto.PublicKey // TODO
	newBlock   *pri.Block
}

func NewMempool(dir string, difficulty uint32) *Mempool {
	os.MkdirAll(dir, 0755)
	os.MkdirAll(filepath.Join(dir, "blocks"), 0755)
	os.MkdirAll(filepath.Join(dir, "wallets"), 0755)
	fmtfilename := filepath.Join(dir, "blocks", "Block%d.dat")

	minerKey, err := crypto.LoadKey(filepath.Join(dir, "wallets", "miner.key"))
	if os.IsNotExist(err) {
		minerKey, err = crypto.NewKey()
		if err != nil {
			panic(err)
		}
		err = minerKey.SaveKey(filepath.Join(dir, "wallets", "miner.key"))
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	pool := &Mempool{
		dir: dir,

		chain: newChain(difficulty),

		publicKey:  minerKey.GetPublicKey(),
		newBlock:   nil,
		pendingTxs: map[pri.HashResult]*pri.Transaction{},
	}

	pool.lock.Lock()
	defer pool.lock.Unlock()

	for i := 1; ; i++ {
		filename := fmt.Sprintf(fmtfilename, i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		block, err := pri.Deserialize(data)
		if err != nil {
			break
		}
		_, is_block := block.(*pri.Block)
		if !is_block {
			break
		}
		err = pool.chain.AppendBlock(block.(*pri.Block))
		if err != nil {
			break
		}
	}

	pool.newBlock = pri.NewBlock(
		pri.Hash(pool.chain.blocks[len(pool.chain.blocks)-1]),
		uint32(len(pool.chain.blocks)),
		pool.publicKey,
		0,
	)

	return pool
}

func (pool *Mempool) Mine() {
	time.Sleep(3 * time.Second)

	for {
		pool.lock.RLock()

		if pool.newBlock == nil {
			panic("mempool.Mempool.Mine: newBlock is nil")
		}

		if !pool.chain.VerifyBlock(pool.newBlock, false) {
			panic("mempool.Mempool.Mine: Invalid block")
		}

		pool.newBlock.RandomizeNonce()
		if pool.newBlock.VerifyDifficulty(len(pool.chain.blocks), pool.chain.difficulty) {
			pool.lock.RUnlock()
			break
		}

		pool.lock.RUnlock()

		time.Sleep(6 * time.Second)
	}
}

func (pool *Mempool) AddTransaction(tx *pri.Transaction) error {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if _, ok := pool.pendingTxs[pri.Hash(tx)]; ok {
		return fmt.Errorf("mempool.Mempool.AddTransaction: Transaction already exists")
	}

	transactions := []pri.Transaction{*tx}
	for _, tx := range pool.pendingTxs {
		transactions = append(transactions, *tx)
	}

	ok, total_tips := pool.chain.VerifyTransactions(transactions)
	if !ok {
		return fmt.Errorf("mempool.Mempool.AddTransaction: Invalid transaction")
	}

	pool.pendingTxs[pri.Hash(tx)] = tx
	pool.newBlock = pri.NewBlock(
		pri.Hash(pool.chain.blocks[len(pool.chain.blocks)-1]),
		uint32(len(pool.chain.blocks)),
		pool.publicKey,
		total_tips,
		transactions...,
	)

	// TODO: shall use the following to replace the previous one
	// pool.newBlock.AddTransaction(*tx)

	return nil
}

func (pool *Mempool) AppendBlock(block *pri.Block) error {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if block == nil {
		block = pool.newBlock
	}

	err := pool.chain.AppendBlock(block)
	if err != nil {
		return err
	}

	// save new block
	pool.saveBlock(block, uint32(len(pool.chain.blocks)-1))

	pool.constructNewBlock()
	return nil
}

func (pool *Mempool) SwitchChain(blocks []*pri.Block, height uint32) error {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if len(blocks)+int(height) <= len(pool.chain.blocks) {
		return fmt.Errorf("mempool.Mempool.SwitchChain: Not a longer chain")
	}

	chain_ := pool.chain.Copy()
	origin_height := uint32(len(chain_.blocks)) - 1
	if height > origin_height+1 {
		return fmt.Errorf("mempool.Mempool.SwitchChain: Invalid height")
	}

	for i := height; i <= origin_height; i++ {
		chain_.RollbackBlock()
	}

	for _, block := range blocks {
		err := chain_.AppendBlock(block)
		if err != nil {
			break
		}
	}

	if len(chain_.blocks) <= len(pool.chain.blocks) {
		return fmt.Errorf("mempool.Mempool.SwitchChain: Not a longer chain")
	}

	pool.chain = chain_
	// now save the new blocks
	for i := height; i < uint32(len(pool.chain.blocks)); i++ {
		pool.saveBlock(pool.chain.blocks[i], i)
	}

	pool.constructNewBlock()

	return nil
}

// You should hold the writer lock before calling this function.
func (pool *Mempool) constructNewBlock() {
	for _, tx := range pool.pendingTxs {
		ok, _ := pool.chain.VerifyTransactions([]pri.Transaction{*tx})
		if !ok {
			delete(pool.pendingTxs, pri.Hash(tx))
		}
	}

	current_transactions := []pri.Transaction{}
	for _, tx := range pool.pendingTxs {
		current_transactions = append(current_transactions, *tx)
	}

	ok, tips := pool.chain.VerifyTransactions(current_transactions)
	if !ok {
		panic("mempool.Mempool.constructNewBlock: Invalid transaction")
	}
	pool.newBlock = pri.NewBlock(
		pri.Hash(pool.chain.blocks[len(pool.chain.blocks)-1]),
		uint32(len(pool.chain.blocks)),
		pool.publicKey,
		tips,
		current_transactions...,
	)
}

func (pool *Mempool) saveBlock(block *pri.Block, height uint32) error {
	filename := filepath.Join(pool.dir, "blocks", fmt.Sprintf("Block%d.dat", height))
	bytes, err := pri.Serialize(block)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filename, bytes, 0644)
	return err
}

func (pool *Mempool) GetLatestInfo() (uint32, *pri.Block) {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return uint32(len(pool.chain.blocks) - 1), pool.chain.blocks[len(pool.chain.blocks)-1]
}

func (pool *Mempool) GetBlock(height uint32) *pri.Block {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	if height >= uint32(len(pool.chain.blocks)) {
		return nil
	}

	return pool.chain.blocks[height]
}

func (pool *Mempool) GetBlockHash(height uint32) pri.HashResult {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	if height >= uint32(len(pool.chain.blocks)) {
		return pri.DEFAULT_HASH_RESULT
	}

	return pri.Hash(pool.chain.blocks[height])
}

func (pool *Mempool) GetTxAmount(ptr pri.TxIn) uint64 {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return pool.chain.txs[ptr.GetTxPtr()].GetTxOuts()[ptr.GetIndex()].GetValue()
}

func (pool *Mempool) ConstructTransaction(send *crypto.PublicKey, recv *crypto.PublicKey, amount uint64, fee uint64) (*pri.Transaction, error) {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	if send == nil || recv == nil {
		return nil, fmt.Errorf("mempool.Mempool.ConstructTransaction: Invalid public key")
	}

	var unspent_amount uint64 = 0
	var unspent_txs []pri.TxIn = []pri.TxIn{}
	var tx_outs []pri.TxOut = []pri.TxOut{}

	for key, tx := range pool.chain.txs {
		unspent := pool.chain.utxos[key]

		for i := 0; i < len(tx.GetTxOuts()); i++ {
			if !unspent[i] {
				continue
			}

			if tx.GetTxOuts()[i].GetPubKey().Equal(send) {
				unspent_amount += tx.GetTxOuts()[i].GetValue()
				unspent_txs = append(unspent_txs, *pri.NewTxIn(key, uint32(i)))
			}

			if unspent_amount >= amount+fee {
				break
			}
		}

		if unspent_amount >= amount+fee {
			break
		}
	}

	if unspent_amount < amount+fee {
		return nil, fmt.Errorf("mempool.Mempool.ConstructTransaction: Insufficient balance")
	}

	if unspent_amount > amount+fee {
		change := unspent_amount - amount - fee
		tx_outs = append(tx_outs, *pri.NewTxOut(change, send))
	}
	tx_outs = append(tx_outs, *pri.NewTxOut(amount, recv))

	tx := pri.NewTx(
		unspent_txs,
		tx_outs,
		nil,
	)

	return tx, nil
}
