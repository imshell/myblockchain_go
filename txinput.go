package main

import "bytes"

type TxInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

func (input *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubkey(input.PubKey)
	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (input *TxInput) CanUnlockOutputWith(unlockData string) bool {
	return input.ScriptSig == unlockData
}
