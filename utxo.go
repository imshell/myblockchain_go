package main

import (
	"log"

	"encoding/hex"

	"github.com/boltdb/bolt"
)

const utxoBucket = "utxo.db"

type UTXOSet struct {
	Blockchain *Blockchain
}

// 把对应的utxo集合单独存储
func (u *UTXOSet) Reindex() {
	db := u.Blockchain.db

	err := db.Update(func(tx *bolt.Tx) error {
		// 每一次都重新创建一个bucket
		err := tx.DeleteBucket([]byte(utxoBucket))
		_, err = tx.CreateBucket([]byte(utxoBucket))

		return err
	})

	if err != nil {
		log.Panicln(err)
	}

	utxo := u.Blockchain.FindUTXO()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		for txid, outs := range utxo {
			key, _ := hex.DecodeString(txid)
			b.Put(key, outs.Serialize())
		}

		return nil
	})
}

// 查找某地址所有可使用的输出，并且拼凑成功对应要支付的金额
func (u *UTXOSet) FindSpendableOutput(pubKeyHash []byte, amount int) (int, map[string][]int) {
	accumulated := 0
	// 把查找到的交易的id和对应的输出索引用key/value保存
	validatedOutput := make(map[string][]int)

	u.Blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txId := hex.EncodeToString(k)
			outputs := DeserializeOutputs(v)

			for outIdx, out := range outputs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					validatedOutput[txId] = append(validatedOutput[txId], outIdx)
				}
			}
		}

		return nil
	})

	return accumulated, validatedOutput
}

// 查找某一个地址的utxo
func (u *UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var utxo []TxOutput

	u.Blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outputs := DeserializeOutputs(v)

			for _, out := range outputs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					utxo = append(utxo, out)
				}
			}
		}
		return nil
	})

	return utxo
}

// 添加一个区块的时候更新utxo
func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		for _, transaction := range block.Transactions {
			if transaction.IsCoinbase() {
				continue
			}

			for _, in := range transaction.Vin {
				updateOps := TxOutputs{}
				opBytes := b.Get(in.Txid)
				outputs := DeserializeOutputs(opBytes)
				for outIdx, out := range outputs.Outputs {
					// 说明此处交易没有被花费
					if outIdx != in.Vout {
						updateOps.Outputs = append(updateOps.Outputs, out)
					}
				}

				if len(updateOps.Outputs) == 0 {
					b.Delete(in.Txid)
				} else {
					b.Put(in.Txid, updateOps.Serialize())
				}
			}

			newOutputs := TxOutputs{}
			for _, out := range transaction.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}
			b.Put(transaction.ID, newOutputs.Serialize())
		}

		return nil
	})

	if err != nil {
		log.Panicln(err)
	}

}
