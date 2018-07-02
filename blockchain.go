package main

import (
	"log"
	"os"

	"fmt"

	"encoding/hex"

	"github.com/boltdb/bolt"
)

const (
	dbFile              = "blockchain.db"
	blockBucket         = "blocks"
	genesisCoinbaseData = "xyb is god"
)

type Blockchain struct {
	tip []byte // 最后一个区块的hash
	db  *bolt.DB
}

func NewBlockchain(address string) *Blockchain {
	if !dbExist() {
		fmt.Println("区块链尚未创建成功")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panicln(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))
		if bucket == nil {
			log.Panicln("区块链不存在")
		}

		tip = bucket.Get([]byte("1"))
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

// 提供给命令行创建一个全新的区块链
func Createblockchain(address string) *Blockchain {
	if dbExist() {
		fmt.Println("区块链已经创建成功，请勿重新创建")
		os.Exit(1)
	}

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

			// 创建coinbase交易
			cbtx := NewCoinbaseTx(address, genesisCoinbaseData)

			// 创世区块
			b := NewGenesisBlock(cbtx)

			// 把对应的区块hash和区块序列化后的数据作为key/value对存储
			err = bucket.Put(b.Hash, b.Serialized())
			if err != nil {
				return err
			}

			// 把最后一块区块的hash保存到数据库
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

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{
		currentHash: bc.tip,
		db:          bc.db,
	}
}

// 获取某地址所有未被花费的交易
func (bc *Blockchain) FindUnSpendTransaction(address string) []*Transaction {
	var unSpendTransaction []*Transaction
	// 该地址所有的输入集合,因为一个输入可以有多个输出，所以定义一个交易id和输出索引的map
	spendTx := make(map[string][]int)

	bci := bc.Iterator()
	for {
		b := bci.Next()

		for _, tx := range b.Transactions {
		Output:
			// 寻找所有的输出
			for outIdx, out := range tx.Vout {
				// 转为字符串id
				txStrId := hex.EncodeToString(tx.ID)

				// 如果属于该地址的输入，则进入判断
				if spendTx[txStrId] != nil {
					for _, voutId := range spendTx[txStrId] {

						// 该输入已被花费，则不会放到 unSpendTransaction 中
						if voutId == outIdx {
							continue Output
						}
					}
				}

				// 使用该地址才能解锁的输出就是该地址拥有的币
				if out.CanBeUnlockedWith(address) {
					unSpendTransaction = append(unSpendTransaction, tx)
				}
			}

			// 必须是非coinbase输入,因为其不会引用任何输出
			if !tx.IsCoinbase() {
				// 寻找所有的输入
				for _, in := range tx.Vin {
					// 是否属于该地址的输入？
					if in.CanUnlockOutputWith(address) {
						txStrId := hex.EncodeToString(in.Txid)
						spendTx[txStrId] = append(spendTx[txStrId], in.Vout)
					}
				}
			}
		}

		if len(b.PrevHash) == 0 {
			break
		}
	}

	return unSpendTransaction
}

// 获取该地址的utxo
func (bc *Blockchain) FindUTXO(address string) []TxOutput {
	var outputs []TxOutput

	unSpendTx := bc.FindUnSpendTransaction(address)
	for _, tx := range unSpendTx {
		for _, out := range tx.Vout {
			// 能被该地址解锁的才能放进utxo
			if out.CanBeUnlockedWith(address) {
				outputs = append(outputs, out)
			}
		}
	}

	return outputs
}

// 查找某地址所有可使用的输出，并且拼凑成功对应要支付的金额
func (bc *Blockchain) FindSpendableOutput(address string, amount int) (int, map[string][]int) {
	unSpendTx := bc.FindUnSpendTransaction(address)
	accumulated := 0
	// 把查找到的交易的id和对应的输出索引用key/value保存
	validatedOutput := make(map[string][]int)

Work:
	for _, tx := range unSpendTx {
		txStrId := hex.EncodeToString(tx.ID)

		for idx, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) && amount > accumulated {
				validatedOutput[txStrId] = append(validatedOutput[txStrId], idx)
				accumulated += out.Value // 凑齐金额
			}

			// 凑齐就退出循环
			if accumulated >= amount {
				break Work
			}
		}
	}

	return accumulated, validatedOutput
}

// 新建一笔交易，也就是新建一个utxo
func (bc *Blockchain) NewUTXOTransaction(from, to string, amount int) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	accumulated, validatedOutput := bc.FindSpendableOutput(from, amount)

	if accumulated < amount {
		log.Panicln("ERROR: 余额不足")
	}

	for id, out := range validatedOutput {
		idx, err := hex.DecodeString(id)

		if err != nil {
			continue
		}

		for outId := range out {
			input := TxInput{
				Txid:      idx,
				Vout:      outId,
				ScriptSig: from, // 标志是这个地址的输入
			}

			inputs = append(inputs, input)
		}
	}

	// 把给对方的这笔交易加进对方的utxo
	outputs = append(outputs, TxOutput{
		Value:        amount,
		ScriptPubKey: to,
	})

	// 找零，把剩余的钱放进自己的utxo
	if accumulated-amount > 0 {
		outputs = append(outputs, TxOutput{
			Value:        accumulated - amount,
			ScriptPubKey: from,
		})
	}

	tx := &Transaction{
		Vin:  inputs,
		Vout: outputs,
	}

	tx.SetID()
	return tx
}

func (bc *Blockchain) MineBlock(txs []*Transaction) {
	var lastHash []byte

	bc.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))
		lastHash = bucket.Get([]byte("1"))
		return nil
	})

	block := NewBlock(txs, lastHash)

	err := bc.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))

		err := bucket.Put([]byte("1"), block.Hash)
		if err != nil {
			return err
		}

		err = bucket.Put(block.Hash, block.Serialized())
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

func (bc *Blockchain) CloseDb() {
	bc.db.Close()
}

// 判断区块链是否已经创建成功
func dbExist() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}
