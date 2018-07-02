package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
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

func NewCoinbaseTx(to, data string) *Transaction {
	// coinbase的输入不会有任何引入输出，因此vout设置为0，id也标记为一个空字节
	txInput := TxInput{
		Txid:      []byte{},
		Vout:      -1,
		ScriptSig: data,
	}

	// coinbase的输出是挖矿获得的奖励，而输出的解锁地址正是传入的这个to的地址，也就是矿工地址
	txOutput := TxOutput{
		Value:        subsidy,
		ScriptPubKey: to,
	}

	tx := &Transaction{Vin: []TxInput{txInput}, Vout: []TxOutput{txOutput}}
	tx.SetID()
	return tx
}
