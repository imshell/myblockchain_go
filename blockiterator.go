package main

import "github.com/boltdb/bolt"

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
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
