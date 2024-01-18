package primitives

import (
	"crypto/sha256"
	"errors"
	"io"

	"os-project/SophiaCoin/pkg/crypto"
)

type TxOut struct {
	value  uint64
	pubKey publicKey
}

func NewTxOut(value uint64, pubKey *crypto.PublicKey) *TxOut {
	return &TxOut{value, publicKey(pubKey.ToBytes())}
}

func (txOut *TxOut) serialize() []byte {
	var result []byte
	result = append(result, uint64ToBytes(txOut.value)...)
	result = append(result, txOut.pubKey[:]...)
	return result
}

func (txOut *TxOut) deserialize(data io.Reader) error {
	var err error
	txOut.value, err = bytesToUint64(data)
	if err != nil {
		return err
	}
	_, err = data.Read(txOut.pubKey[:])
	if err == io.EOF {
		return errors.New("primitives.TxOut.Deserialize: Unexpected EOF")
	}
	return err
}

func (txOut *TxOut) hash() HashResult {
	return sha256.Sum256(txOut.serialize())
}

func (txOut *TxOut) GetValue() uint64 {
	return txOut.value
}

func (txOut *TxOut) GetPubKey() *crypto.PublicKey {
	key, _ := crypto.FromBytes(txOut.pubKey[:])
	return key
}
