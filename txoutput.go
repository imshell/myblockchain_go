package main

import "bytes"

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
