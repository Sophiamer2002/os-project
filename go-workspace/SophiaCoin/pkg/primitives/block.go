package primitives

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os-project/SophiaCoin/pkg/crypto"
	"time"
)

type BlockHeader struct {
	timestamp  uint64
	nonce      uint32
	prevBlock  HashResult
	merkleRoot HashResult
}

type Block struct {
	header       BlockHeader
	transactions []Transaction
	tree         *merkleTree // lazy initialization
}

func GetGenesisBlock() *Block {
	return &Block{
		header: BlockHeader{
			timestamp:  uint64(time.Date(2002, time.November, 11, 18, 12, 0, 0, time.UTC).Unix()),
			nonce:      0xdeadbeef,
			prevBlock:  DEFAULT_HASH_RESULT,
			merkleRoot: DEFAULT_HASH_RESULT,
		},
		transactions: []Transaction{},
		tree:         nil,
	}
}

func NewBlock(
	prevBlock HashResult,
	height uint32,
	minerPubkey *crypto.PublicKey, // miner's public key, used to construct coinbase transaction
	total_tips uint64,
	transactions ...Transaction,
) *Block {
	coinbase := Transaction{
		txIns: []TxIn{{
			txPtr: DEFAULT_HASH_RESULT,
			index: height,
		}},
		txOuts: []TxOut{{
			value:  total_tips + MINER_REWARD,
			pubKey: publicKey(minerPubkey.ToBytes()),
		}},
		signatures: []signature{},
	}

	b := &Block{
		header: BlockHeader{
			timestamp:  uint64(time.Now().Unix()),
			nonce:      0xdeadbeef,
			prevBlock:  prevBlock,
			merkleRoot: DEFAULT_HASH_RESULT,
		},
		transactions: append([]Transaction{coinbase}, transactions...),
		tree:         nil,
	}
	b.constructMerkleTree()
	b.header.merkleRoot = b.tree.root()
	return b
}

func (bh *BlockHeader) serialize() []byte {
	var result []byte
	result = append(result, uint64ToBytes(bh.timestamp)...)
	result = append(result, uint32ToBytes(bh.nonce)...)
	result = append(result, bh.prevBlock[:]...)
	result = append(result, bh.merkleRoot[:]...)
	return result
}

func (bh *BlockHeader) deserialize(data io.Reader) error {
	var err error
	bh.timestamp, err = bytesToUint64(data)
	if err != nil {
		return err
	}
	bh.nonce, err = bytesToUint32(data)
	if err != nil {
		return err
	}
	_, err = data.Read(bh.prevBlock[:])
	if err == io.EOF {
		return errors.New("primitives.BlockHeader.deserialize: Unexpected EOF")
	} else if err != nil {
		return err
	}
	_, err = data.Read(bh.merkleRoot[:])
	if err == io.EOF {
		return errors.New("primitives.BlockHeader.deserialize: Unexpected EOF")
	} else if err != nil {
		return err
	}
	return nil
}

func (bh *BlockHeader) hash() HashResult {
	return sha256.Sum256(bh.serialize())
}

func (bh *BlockHeader) GetMerkleRoot() HashResult {
	return bh.merkleRoot
}

func (bh *BlockHeader) GetTimestamp() uint64 {
	return bh.timestamp
}

func (b *Block) serialize() []byte {
	var result []byte
	result = append(result, b.header.serialize()...)
	result = append(result, uint32ToBytes(uint32(len(b.transactions)))...)
	for _, tx := range b.transactions {
		result = append(result, tx.serialize()...)
	}
	return result
}

func (b *Block) deserialize(data io.Reader) error {
	err := b.header.deserialize(data)
	if err != nil {
		return err
	}

	b.transactions = []Transaction{}
	txLen, err := bytesToUint32(data)
	if err != nil {
		return err
	}
	for i := 0; i < int(txLen); i++ {
		tx := Transaction{}
		err = tx.deserialize(data)
		if err != nil {
			return err
		}
		b.transactions = append(b.transactions, tx)
	}

	return nil
}

func (b *Block) hash() HashResult {
	return b.header.hash()
}

func (b *Block) AddTransaction(tx ...Transaction) {
	b.transactions = append(b.transactions, tx...)
	if b.tree == nil {
		b.constructMerkleTree()
	} else {
		hashes := make([]HashResult, 0, len(tx))
		for _, tx := range tx {
			hashes = append(hashes, tx.hash())
		}
		b.tree.append(hashes...)
	}
	b.header.merkleRoot = b.tree.root()
}

func (b *Block) constructMerkleTree() {
	hashes := make([]HashResult, 0, len(b.transactions))
	for _, tx := range b.transactions {
		hashes = append(hashes, tx.hash())
	}
	b.tree = newMerkleTree(hashes)
}

func (b *Block) GetTransactions() []Transaction {
	return b.transactions
}

func (b *Block) GetHeader() *BlockHeader {
	return &b.header
}

func (b *Block) RandomizeNonce() {
	var err error
	four_bytes := crypto.RandBytes(4)
	b.header.nonce, err = bytesToUint32(bytes.NewReader(four_bytes))
	if err != nil {
		panic(err)
	}
}
