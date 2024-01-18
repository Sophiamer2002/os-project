package primitives

// This file overloads the "%v" format specifier for the primitives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

func jsonDump(s string) string {
	var str bytes.Buffer
	_ = json.Indent(&str, []byte(s), "", "    ")
	return str.String()
}

func (hash HashResult) String() string {
	return fmt.Sprintf("\"Hash+0x%x\"", hash[:])
}

func (pubKey publicKey) String() string {
	return fmt.Sprintf("\"PublicKey+0x%x\"", pubKey[:])
}

func (txIn TxIn) String() string {
	s := fmt.Sprintf("{\"txPtr\": %v, \"index\": %v}", txIn.txPtr, txIn.index)
	_, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return s
}

func (txOut TxOut) String() string {
	s := fmt.Sprintf("{\"amount\": %v, \"address\": %v}", txOut.value, txOut.pubKey)
	return jsonDump(s)
}

func (tx Transaction) String() string {
	txIn_str := ""
	for _, txIn := range tx.txIns {
		txIn_str += fmt.Sprintf("%v, ", txIn)
	}
	txOut_str := ""
	for _, txOut := range tx.txOuts {
		txOut_str += fmt.Sprintf("%v, ", txOut)
	}
	signatures_str := ""
	for _, sig := range tx.signatures {
		signatures_str += fmt.Sprintf("\"Signature+0x%x\", ", sig[:])
	}
	if len(txIn_str) == 0 {
		txIn_str = ", "
	}
	if len(txOut_str) == 0 {
		txOut_str = ", "
	}
	if len(signatures_str) == 0 {
		signatures_str = ", "
	}
	s := fmt.Sprintf("{\"hash\": %v,\"txIns\": [%v],\"txOuts\": [%v],\"signatures\": [%v]}",
		tx.hash(), txIn_str[:len(txIn_str)-2], txOut_str[:len(txOut_str)-2], signatures_str[:len(signatures_str)-2])
	return jsonDump(s)
}

func (header BlockHeader) String() string {
	time := time.Unix(int64(header.timestamp), 0)
	s := fmt.Sprintf("{\"hash\": %v,\"timestamp\": %v,\"time\": \"%v\", \"nonce\": %v,\"prevBlock\": %v,\"merkleRoot\": %v}",
		header.hash(), header.timestamp, time, header.nonce, header.prevBlock, header.merkleRoot)
	return jsonDump(s)
}

func (block Block) String() string {
	tx_str := ""
	for _, tx := range block.transactions {
		tx_str += fmt.Sprintf("%v, ", tx)
	}

	if len(tx_str) == 0 {
		tx_str = ", "
	}
	s := fmt.Sprintf("{\"header\": %v,\"transactions\": [%v]}", block.header, tx_str[:len(tx_str)-2])
	return jsonDump(s)
}
