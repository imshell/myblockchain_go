package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

func NewUtxo(amount int, address string) TxOutput {
	output := TxOutput{Value: amount}
	output.Lock([]byte(address))
	return output
}

func (output *TxOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)            // 先解码
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4] // 去掉checksum和version 拿出公钥hash
	output.PubKeyHash = pubKeyHash
}

func (output *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(pubKeyHash, output.PubKeyHash) == 0
}

type TxOutputs struct {
	Outputs []TxOutput
}

func (outputs *TxOutputs) Serialize() []byte {
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(outputs)

	if err != nil {
		log.Panicln(err)
	}

	return buff.Bytes()
}

func DeserializeOutputs(opByte []byte) TxOutputs {
	outputs := TxOutputs{}

	decoder := gob.NewDecoder(bytes.NewReader(opByte))
	err := decoder.Decode(&outputs)

	if err != nil {
		log.Panicln(err)
	}

	return outputs
}
