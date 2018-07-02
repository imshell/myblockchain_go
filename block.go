package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp int64
	Data      []byte
	Hash      []byte
	PrevHash  []byte
	Nonce     int
}

func NewBlock(data string, prevHash []byte) *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		Data:      []byte(data),
		PrevHash:  prevHash,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash[:]

	return block
}

func NewGenesisBlock() *Block {
	return NewBlock("老子最大", []byte{})
}

func (b *Block) Serialized() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)

	if err != nil {
		log.Panicln(err)
	}

	return result.Bytes()
}

func Deserialize(data []byte) *Block {
	block := &Block{}
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(block)
	if err != nil {
		log.Panicln(err)
	}

	return block
}
