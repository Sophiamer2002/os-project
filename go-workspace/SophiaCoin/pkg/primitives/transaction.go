package primitives

import (
	"crypto/sha256"
	"errors"
	"io"
	"os-project/SophiaCoin/pkg/crypto"
)

type Transaction struct {
	txIns      []TxIn
	txOuts     []TxOut
	signatures []signature
}

func NewTx(txIns []TxIn, txOuts []TxOut, signatures []signature) *Transaction {
	return &Transaction{txIns, txOuts, signatures}
}

func (tx *Transaction) serialize() []byte {
	var result []byte
	result = append(result, uint32ToBytes(uint32(len(tx.txIns)))...)
	for _, txIn := range tx.txIns {
		result = append(result, txIn.serialize()...)
	}
	result = append(result, uint32ToBytes(uint32(len(tx.txOuts)))...)
	for _, txOut := range tx.txOuts {
		result = append(result, txOut.serialize()...)
	}
	result = append(result, uint32ToBytes(uint32(len(tx.signatures)))...)
	for _, signature := range tx.signatures {
		result = append(result, uint32ToBytes(uint32(len(signature)))...)
		result = append(result, signature[:]...)
	}

	return result
}

func (tx *Transaction) deserialize(data io.Reader) error {
	tx.txIns = []TxIn{}
	tx.txOuts = []TxOut{}
	tx.signatures = []signature{}

	txInsLen, err := bytesToUint32(data)
	if err != nil {
		return err
	}
	for i := 0; i < int(txInsLen); i++ {
		TxIn := TxIn{}
		err = TxIn.deserialize(data)
		if err != nil {
			return err
		}
		tx.txIns = append(tx.txIns, TxIn)
	}

	txOutsLen, err := bytesToUint32(data)
	if err == io.EOF {
		return errors.New("primitives.Transaction.deserialize: Unexpected EOF")
	}
	for i := 0; i < int(txOutsLen); i++ {
		TxOut := TxOut{}
		err = TxOut.deserialize(data)
		if err != nil {
			return err
		}
		tx.txOuts = append(tx.txOuts, TxOut)
	}

	signaturesLen, err := bytesToUint32(data)
	if err == io.EOF {
		return errors.New("primitives.Transaction.deserialize: Unexpected EOF")
	}
	for i := 0; i < int(signaturesLen); i++ {
		length, err := bytesToUint32(data)
		if err != nil {
			return err
		}
		signature := make([]byte, length)
		data.Read(signature)
		tx.signatures = append(tx.signatures, signature)
	}

	return nil
}

func (tx *Transaction) hash() HashResult {
	return sha256.Sum256(tx.serialize())
}

func (tx *Transaction) GetTxIns() []TxIn {
	return tx.txIns
}

func (tx *Transaction) GetTxOuts() []TxOut {
	return tx.txOuts
}

// The function returns whether the transaction relates to the public key
// in its txins if isIn is true, or relates to the public key in its txouts
// otherwise. It returns a list of indices of txins or txouts that relate to
// the public key.
func (tx *Transaction) RelatesTo(pubkey crypto.PublicKey, isIn bool) []int {
	pubkeyBytes := pubkey.ToBytes()
	ret := []int{}
	if isIn && len(tx.signatures) > 0 {
		for i, txIn := range tx.txIns {
			raw := &Transaction{
				txIns:      []TxIn{txIn},
				txOuts:     tx.txOuts,
				signatures: []signature{},
			}
			raw_bytes := sha256.Sum256(raw.serialize())
			if pubkey.Verify(raw_bytes[:], tx.signatures[i][:]) {
				ret = append(ret, i)
			}
		}
	} else if !isIn {
		for i, txOut := range tx.txOuts {
			if publicKey(pubkeyBytes) == txOut.pubKey {
				ret = append(ret, i)
			}
		}
	}

	return ret
}
