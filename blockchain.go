package main

import (
	"log"

	"github.com/boltdb/bolt"
)

const (
	dbFile      = "blockchain.db"
	blockBucket = "blocks"
)

type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panicln(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))
		if bucket == nil {
			bucket, err := tx.CreateBucket([]byte(blockBucket))
			if err != nil {
				return err
			}

			b := NewGenesisBlock()

			err = bucket.Put(b.Hash, b.Serialized())
			if err != nil {
				return err
			}

			bucket.Put([]byte("1"), b.Hash)
			tip = b.Hash
		} else {
			tip = bucket.Get([]byte("1"))
		}
		return nil
	})

	if err != nil {
		log.Panicln(err)
	}

	return &Blockchain{
		tip: tip,
		db:  db,
	}
}

func (bc *Blockchain) AddBlock(data string) {
	err := bc.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))

		lastHash := bucket.Get([]byte("1"))

		block := NewBlock(data, lastHash)

		err := bucket.Put(block.Hash, block.Serialized())
		if err != nil {
			return err
		}

		err = bucket.Put([]byte("1"), block.Hash)
		if err != nil {
			return err
		}

		bc.tip = block.Hash

		return nil
	})

	if err != nil {
		log.Panicln(err)
	}
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{
		currentHash: bc.tip,
		db:          bc.db,
	}
}

func (bci *BlockchainIterator) Next() *Block {
	var block *Block
	bci.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))
		encode := bucket.Get(bci.currentHash)
		block = Deserialize(encode)
		return nil
	})

	bci.currentHash = block.PrevHash
	return block
}
