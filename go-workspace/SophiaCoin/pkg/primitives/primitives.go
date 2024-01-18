package primitives

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

type Serializable interface {
	serialize() []byte
	deserialize(io.Reader) error
	hash() HashResult
}

type signature []byte
type publicKey [91]byte
type HashResult [32]byte

const (
	BLOCK_HEADER uint32 = iota
	BLOCK
	TX
	TX_IN
	TX_OUT
)

var (
	// DO NOT CHANGE THE VALUES
	DEFAULT_HASH_RESULT HashResult = sha256.Sum256([]byte{})
	MINER_REWARD                   = uint64(1024)
)

func Serialize(data Serializable) ([]byte, error) {
	var result []byte
	switch data.(type) {
	case *BlockHeader:
		result = append(result, uint32ToBytes(BLOCK_HEADER)...)
	case *Block:
		result = append(result, uint32ToBytes(BLOCK)...)
	case *Transaction:
		result = append(result, uint32ToBytes(TX)...)
	case *TxIn:
		result = append(result, uint32ToBytes(TX_IN)...)
	case *TxOut:
		result = append(result, uint32ToBytes(TX_OUT)...)
	default:
		return nil, errors.New("primitives.Serialize: Unknown data type")
	}

	result = append(result, data.serialize()...)
	return result, nil
}

func Deserialize(data []byte) (Serializable, error) {
	if len(data) < 4 {
		return nil, errors.New("primitives.Deserialize: Invalid data length")
	}

	r := bytes.NewReader(data)

	dataType, err := bytesToUint32(r)
	if err != nil {
		return nil, err
	}
	switch dataType {
	case BLOCK_HEADER:
		var blockHeader BlockHeader
		err := blockHeader.deserialize(r)
		return &blockHeader, err
	case BLOCK:
		var block Block
		err := block.deserialize(r)
		return &block, err
	case TX:
		var tx Transaction
		err := tx.deserialize(r)
		return &tx, err
	case TX_IN:
		var txIn TxIn
		err := txIn.deserialize(r)
		return &txIn, err
	case TX_OUT:
		var txOut TxOut
		err := txOut.deserialize(r)
		return &txOut, err
	default:
		return nil, errors.New("primitives.Deserialize: Unknown data type")
	}
}

func Hash(data Serializable) HashResult {
	return data.hash()
}

func uint32ToBytes(n uint32) []byte {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, n)
	return result
}

func uint64ToBytes(n uint64) []byte {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, n)
	return result
}

func bytesToUint32(data io.Reader) (uint32, error) {
	var result uint32
	err := binary.Read(data, binary.LittleEndian, &result)
	return result, err
}

func bytesToUint64(data io.Reader) (uint64, error) {
	var result uint64
	err := binary.Read(data, binary.LittleEndian, &result)
	return result, err
}
