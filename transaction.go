package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

const subsidy = 10

type TxInput struct {
	Txid      []byte
	Vout      int
	ScriptSig string
}

type TxOutput struct {
	Value        int
	ScriptPubKey string
}

type Transaction struct {
	ID   []byte
	Vin  []TxInput
	Vout []TxOutput
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vout[0] == -1
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

func (input *TxInput) CanUnlockOutputWith(unlockData string) bool {
	return input.ScriptSig == unlockData
}

func (output *TxOutput) CanBeUnlockedWith(unlockData string) bool {
	return output.ScriptPubKey == unlockData
}
