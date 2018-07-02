package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"log"
	"math/big"
)

const subsidy = 10

type Transaction struct {
	ID   []byte
	Vin  []TxInput
	Vout []TxOutput
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func (tx *Transaction) SetID() {
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(tx)
	if err != nil {
		log.Panicln(err)
	}

	hash := sha256.Sum256(buffer.Bytes())
	tx.ID = hash[:]
}

// 对交易输入进行签名
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	// 去除signature和pubkey的一个副本
	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		// 拿出本次输入对应的输出交易
		prevTx := prevTxs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		// 把本次输入的pubkey置为引用输出的pubkeyhash
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		// 哈希本次交易获取一个txid
		txCopy.ID = txCopy.Hash()
		// 为了不影响下一次迭代
		txCopy.Vin[inID].PubKey = nil

		// 签名
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		if err != nil {
			log.Panicln(err)
		}

		signature := append(r.Bytes(), s.Bytes()...)
		tx.Vin[inID].Signature = signature
	}
}

// 检查签名
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Vin {
		inputs = append(inputs, TxInput{in.Txid, in.Vout, nil, nil})
	}

	for _, out := range tx.Vout {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	t := Transaction{tx.ID, inputs, outputs}

	return t
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txcopy := *tx
	txcopy.ID = []byte{}

	hash = sha256.Sum256(txcopy.Serialize())

	return hash[:]
}

func (tx Transaction) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)

	if err != nil {
		log.Panicln(err)
	}

	return buff.Bytes()
}

func NewCoinbaseTx(to, data string) *Transaction {
	// coinbase的输入不会有任何引入输出，因此vout设置为0，id也标记为一个空字节
	txInput := TxInput{
		Txid:      []byte{},
		Vout:      -1,
		Signature: nil,
		PubKey:    []byte(data),
	}

	// coinbase的输出是挖矿获得的奖励，而输出的解锁地址正是传入的这个to的地址，也就是矿工地址
	txOutput := NewUtxo(subsidy, to)

	tx := &Transaction{Vin: []TxInput{txInput}, Vout: []TxOutput{txOutput}}
	tx.ID = tx.Hash()
	return tx
}
