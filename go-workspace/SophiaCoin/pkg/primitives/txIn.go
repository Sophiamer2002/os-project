package primitives

import (
	"crypto/sha256"
	"errors"
	"io"
)

type TxIn struct {
	txPtr HashResult
	index uint32
}

func NewTxIn(txPtr HashResult, index uint32) *TxIn {
	return &TxIn{txPtr, index}
}

func (txIn *TxIn) serialize() []byte {
	var result []byte
	result = append(result, txIn.txPtr[:]...)
	result = append(result, uint32ToBytes(txIn.index)...)
	return result
}

func (txIn *TxIn) deserialize(data io.Reader) error {
	_, err := data.Read(txIn.txPtr[:])
	if err == io.EOF {
		return errors.New("primitives.TxIn.deserialize: Unexpected EOF")
	} else if err != nil {
		return err
	}
	txIn.index, err = bytesToUint32(data)
	if err != nil {
		return err
	}
	return err
}

func (txIn *TxIn) hash() HashResult {
	return sha256.Sum256(txIn.serialize())
}

func (txIn *TxIn) GetTxPtr() HashResult {
	return txIn.txPtr
}

func (txIn *TxIn) GetIndex() uint32 {
	return txIn.index
}
