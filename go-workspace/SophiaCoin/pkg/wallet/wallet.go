package wallet

import (
	"encoding/hex"
	"fmt"
	"os"
	"os-project/SophiaCoin/pkg/crypto"
	pri "os-project/SophiaCoin/pkg/primitives"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

type Wallet struct {
	dir  string
	lock sync.RWMutex

	keys       map[string]*crypto.Key
	pubs       map[string]*crypto.PublicKey
	headers    []*pri.BlockHeader
	tx_history dataframe.DataFrame
}

type TxRecord struct {
	BlockHeight int
	blockhash   pri.HashResult
	TxHash      string
	TxIdx       int
	IsTxIn      bool
	InOutIdx    int
	Amount      int
	Address     string
	merkleProof []pri.HashResult
}

var (
	// Fields for the tx_history dataframe
	BlockHeight = "BlockHeight"
	TxHash      = "TxHash"
	TxIdx       = "TxIdx"
	IsTxIn      = "IsTxIn"
	InOutIdx    = "InOutIdx"
	Amount      = "Amount"
	Address     = "Address"
)

func NewWallet(dir string) *Wallet {
	w := &Wallet{
		dir: dir,

		keys:    map[string]*crypto.Key{},
		pubs:    map[string]*crypto.PublicKey{},
		headers: []*pri.BlockHeader{pri.GetGenesisBlock().GetHeader()},
		tx_history: dataframe.New(
			series.New([]int{}, series.Int, BlockHeight),
			series.New([]string{}, series.String, TxHash),
			series.New([]int{}, series.Int, TxIdx),
			series.New([]bool{}, series.Bool, IsTxIn),
			series.New([]int{}, series.Int, InOutIdx),
			series.New([]int{}, series.Int, Amount),
			series.New([]string{}, series.String, Address),
		),
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	os.MkdirAll(filepath.Join(dir, "wallets"), 0755)
	entries, err := os.ReadDir(filepath.Join(dir, "wallets"))
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		key, err := crypto.LoadKey(filepath.Join(dir, "wallets", entry.Name()))
		if err != nil {
			continue
		}
		w.keys[strings.Split(entry.Name(), ".")[0]] = key
	}

	os.Mkdir(filepath.Join(dir, "pubkeys"), 0755)
	entries, err = os.ReadDir(filepath.Join(dir, "pubkeys"))
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		b, _ := os.ReadFile(filepath.Join(dir, "pubkeys", entry.Name()))
		key, err := crypto.FromBytes(b)
		if err != nil {
			continue
		}
		w.pubs[strings.Split(entry.Name(), ".")[0]] = key
	}

	return w
}

func NewRecord(
	blockHeight int,
	blockHash pri.HashResult,
	txHash pri.HashResult,
	txIdx int,
	isTxIn bool,
	inOutIdx int,
	amount int,
	address string,
	merkleProof []byte,
) TxRecord {
	return TxRecord{
		BlockHeight: blockHeight,
		blockhash:   blockHash,
		TxHash:      fmt.Sprintf("0x%x", txHash[:]),
		TxIdx:       txIdx,
		IsTxIn:      isTxIn,
		InOutIdx:    inOutIdx,
		Amount:      amount,
		Address:     address,
		merkleProof: nil, // TODO
	}
}

func (w *Wallet) UpdateHeaders(from uint32, headers ...*pri.BlockHeader) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if from > uint32(len(w.headers)) || from == 0 {
		panic("Wallet.UpdateHeaders: Invalid starting header")
	}

	if from+uint32(len(headers)) <= uint32(len(w.headers)) {
		return fmt.Errorf("Wallet.UpdateHeaders: Not a longer chain")
	}

	// Now check if the headers are valid
	// Since we are not checking the transactions,
	// we only check the chain structure

	for i := 1; i < len(headers); i++ {
		if !headers[i].VerifyPreviousHash(pri.Hash(headers[i-1])) {
			return fmt.Errorf("Wallet.UpdateHeaders: Invalid chain")
		}
	}

	if !headers[0].VerifyPreviousHash(pri.Hash(w.headers[from-1])) {
		return fmt.Errorf("Wallet.UpdateHeaders: Invalid chain")
	}

	w.headers = append(w.headers[:from], headers...)
	w.tx_history = w.tx_history.Filter(
		dataframe.F{
			Colname:    BlockHeight,
			Comparator: series.Less,
			Comparando: int(from),
		},
	)
	return nil
}

func (w *Wallet) GetHeaderHash(height uint32) pri.HashResult {
	w.lock.RLock()
	defer w.lock.RUnlock()
	if height >= uint32(len(w.headers)) {
		return pri.DEFAULT_HASH_RESULT
	}
	return pri.Hash(w.headers[height])
}

func (w *Wallet) AddTxRecords(records ...*TxRecord) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// append only valid records
	for _, record := range records {
		if record.BlockHeight >= len(w.headers) {
			continue
		}
		if !pri.VerifyProof(
			w.headers[record.BlockHeight].GetMerkleRoot(),
			record.merkleProof,
			toHash(record.TxHash),
			record.TxIdx,
		) {
			continue
		}

		if record.blockhash != pri.Hash(w.headers[record.BlockHeight]) {
			continue
		}

		df := w.tx_history.Filter(
			dataframe.F{
				Colname:    TxHash,
				Comparator: series.Eq,
				Comparando: record.TxHash,
			},
		).Filter(
			dataframe.F{
				Colname:    TxIdx,
				Comparator: series.Eq,
				Comparando: record.TxIdx,
			},
		).Filter(
			dataframe.F{
				Colname:    IsTxIn,
				Comparator: series.Eq,
				Comparando: record.IsTxIn,
			},
		).Filter(
			dataframe.F{
				Colname:    InOutIdx,
				Comparator: series.Eq,
				Comparando: record.InOutIdx,
			},
		)

		// If the record already exists, skip
		if df.Nrow() != 0 {
			continue
		}

		w.tx_history = w.tx_history.RBind(
			dataframe.LoadStructs([]TxRecord{*record}),
		)
	}
}

func (w *Wallet) GetBalance() *map[string]int {
	w.lock.RLock()
	defer w.lock.RUnlock()
	balance := map[string]int{}
	for name := range w.keys {
		balance[name] = 0
	}

	if w.tx_history.Nrow() == 0 {
		return &balance
	}

	df := w.tx_history.GroupBy(IsTxIn, Address).Aggregation(
		[]dataframe.AggregationType{dataframe.Aggregation_SUM},
		[]string{Amount},
	)

	if df.Err != nil {
		panic(df.Err)
	}

	names := df.Col(Address).Records()

	expense, err := df.Col(Amount + "_SUM").Int()
	if err != nil {
		panic(err)
	}

	isTxIn, err := df.Col(IsTxIn).Bool()
	if err != nil {
		panic(err)
	}

	for i := 0; i < df.Nrow(); i++ {
		if _, ok := balance[names[i]]; !ok {
			panic("Wallet.GetBalance: Invalid address")
		}
		if isTxIn[i] {
			balance[names[i]] -= expense[i]
		} else {
			balance[names[i]] += expense[i]
		}
	}

	return &balance
}

func toHash(s any) pri.HashResult {
	switch s := s.(type) {
	case pri.HashResult:
		return s
	default:
		str := fmt.Sprintf("%v", s) // "0x123456" -> "123456"
		if strings.HasPrefix(str, "0x") && len(str) == 66 {
			b, err := hex.DecodeString(str[2:])
			if err != nil {
				panic(err)
			}
			return pri.HashResult(b)
		}
		panic("toHash: Invalid type")
	}
}

func (w *Wallet) GetLatestInfo() (uint32, pri.HashResult) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return uint32(len(w.headers)) - 1, pri.Hash(w.headers[len(w.headers)-1])
}

func (w *Wallet) GetSelfAddress() map[string][]byte {
	w.lock.RLock()
	defer w.lock.RUnlock()
	result := map[string][]byte{}
	for name, key := range w.keys {
		result[name] = key.GetPublicKey().ToBytes()
	}
	return result
}

func (w *Wallet) GetKnownAddress() map[string][]byte {
	w.lock.RLock()
	defer w.lock.RUnlock()
	result := map[string][]byte{}
	for name, key := range w.pubs {
		result[name] = key.ToBytes()
	}

	return result
}

func (w *Wallet) GetTxHistory() *dataframe.DataFrame {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return &w.tx_history
}

func (w *Wallet) SignTransaction(tx *pri.Transaction, addr string) error {
	w.lock.RLock()
	defer w.lock.RUnlock()

	key := w.keys[addr]
	if key == nil {
		return fmt.Errorf("Wallet.SignTransaction: Invalid address")
	}

	tx.Sign(key)

	pubs := []*crypto.PublicKey{}
	for range tx.GetTxIns() {
		pubs = append(pubs, key.GetPublicKey())
	}

	if !tx.VerifySignature(pubs) {
		return fmt.Errorf("Wallet.SignTransaction: Invalid signature")
	}

	return nil
}

func (w *Wallet) NewKey(name string) error {
	// check if the name is valid: can only contain alphanumeric characters, or underscore
	for _, c := range name {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
			continue
		}
		return fmt.Errorf("Wallet.NewKey: Invalid name")
	}

	if _, ok := w.keys[name]; ok {
		return fmt.Errorf("Wallet.NewKey: Key already exists")
	}

	w.lock.Lock()
	defer w.lock.Unlock()

	newKey, err := crypto.NewKey()
	if err != nil {
		return fmt.Errorf("Wallet.NewKey: %v", err)
	}

	if err := newKey.SaveKey(filepath.Join(w.dir, "wallets", name+".key")); err != nil {
		return fmt.Errorf("Wallet.NewKey: %v", err)
	}

	w.keys[name] = newKey
	return nil
}

func (w *Wallet) NewPubAddress(name string, addr []byte) error {
	// check if the name is valid: can only contain alphanumeric characters, or underscore
	for _, c := range name {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
			continue
		}
		return fmt.Errorf("Wallet.NewPubAddress: Invalid name")
	}

	if _, ok := w.pubs[name]; ok {
		return fmt.Errorf("Wallet.NewPubAddress: Address already exists")
	}

	w.lock.Lock()
	defer w.lock.Unlock()

	newPub, err := crypto.FromBytes(addr)
	if err != nil {
		return fmt.Errorf("Wallet.NewPubAddress: %v", err)
	}

	err = os.WriteFile(filepath.Join(w.dir, "pubkeys", name+".pub"), addr, 0644)
	if err != nil {
		return fmt.Errorf("Wallet.NewPubAddress: %v", err)
	}

	w.pubs[name] = newPub
	return nil
}

func (w *Wallet) GetPubAddress() map[string][]byte {
	w.lock.RLock()
	defer w.lock.RUnlock()
	result := map[string][]byte{}
	for name, key := range w.pubs {
		result[name] = key.ToBytes()
	}
	return result
}

func (w *Wallet) GetBill(start time.Time, end time.Time, addr []string) *dataframe.DataFrame {
	// first binary search the starting and ending block number

	w.lock.RLock()
	defer w.lock.RUnlock()

	min, max := 0, len(w.headers)-1
	for min < max {
		mid := (min + max) / 2
		if int64(w.headers[mid].GetTimestamp()) < start.Unix() {
			min = mid + 1
		} else {
			max = mid
		}
	}

	startBlock := min

	min, max = 0, len(w.headers)-1
	for min < max {
		mid := (min + max + 1) / 2
		if int64(w.headers[mid].GetTimestamp()) <= end.Unix() {
			min = mid
		} else {
			max = mid - 1
		}
	}

	endBlock := min

	// now we have the starting and ending block number
	// we can filter the tx_history dataframe

	df := w.tx_history.Filter(
		dataframe.F{
			Colname:    BlockHeight,
			Comparator: series.GreaterEq,
			Comparando: startBlock,
		},
	).Filter(
		dataframe.F{
			Colname:    BlockHeight,
			Comparator: series.LessEq,
			Comparando: endBlock,
		},
	)

	filters := []dataframe.F{}
	for _, a := range addr {
		filters = append(filters, dataframe.F{
			Colname:    Address,
			Comparator: series.Eq,
			Comparando: a,
		})
	}

	if len(filters) != 0 {
		df = df.Filter(filters...)
	}

	df = df.Arrange(dataframe.Sort(BlockHeight))

	return &df
}
