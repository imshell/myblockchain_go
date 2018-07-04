package main

import (
	"log"
	"os"

	"fmt"

	"encoding/hex"

	"bytes"
	"errors"

	"crypto/ecdsa"

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

func (bc *Blockchain) GetBestHeight() int {
	var lastHeight int

	bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockBucket))
		blockData := b.Get([]byte("1"))
		lastBlock := Deserialize(blockData)
		lastHeight = lastBlock.Height
		return nil
	})

	return lastHeight
}

// 获取某地址所有未被花费的交易
func (bc *Blockchain) FindUnSpendTransaction(pubKeyHash []byte) []*Transaction {
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
				if out.IsLockedWithKey(pubKeyHash) {
					unSpendTransaction = append(unSpendTransaction, tx)
				}
			}

			// 必须是非coinbase输入,因为其不会引用任何输出
			if !tx.IsCoinbase() {
				// 寻找所有的输入
				for _, in := range tx.Vin {
					// 是否属于该地址的输入？
					if in.UsesKey(pubKeyHash) {
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

func (bc *Blockchain) FindUTXO() map[string]TxOutputs {
	utxo := make(map[string]TxOutputs)
	spendTx := make(map[string][]int)

	bci := bc.Iterator()

	for {
		b := bci.Next()

		for _, tx := range b.Transactions {
			txId := hex.EncodeToString(tx.ID)
		Output:
			for outIdx, out := range tx.Vout {
				// 寻找到对应交易
				if spendTx[txId] != nil {
					for _, voutId := range spendTx[txId] {
						// 如果发现已花费输出引用了改输出的索引，那么认定该输出是已花费输出
						if voutId == outIdx {
							continue Output
						}
					}
				}
				outs := utxo[txId]
				outs.Outputs = append(outs.Outputs, out)
				utxo[txId] = outs
			}

			if !tx.IsCoinbase() {
				// 把所有已经花费的输出都记录进去
				for _, in := range tx.Vin {
					tid := hex.EncodeToString(in.Txid)
					spendTx[tid] = append(spendTx[tid], in.Vout)
				}
			}
		}

		if len(b.PrevHash) == 0 {
			break
		}
	}

	return utxo
}

// 新建一笔交易，也就是新建一个utxo
func (bc *Blockchain) NewUTXOTransaction(from, to string, amount int) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	// 获取钱包
	wallets, err := NewWallets()
	if err != nil {
		log.Panicln(err)
	}

	fromWallet := wallets.GetWallet(from)
	pubKeyHash := HashPubkey(fromWallet.PublicKey)

	utxo := UTXOSet{bc}
	accumulated, validatedOutput := utxo.FindSpendableOutput(pubKeyHash, amount)

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
				Signature: nil,
				PubKey:    fromWallet.PublicKey,
			}

			inputs = append(inputs, input)
		}
	}

	// 把给对方的这笔交易加进对方的utxo
	outputs = append(outputs, NewUtxo(amount, to))

	// 找零，把剩余的钱放进自己的utxo
	if accumulated-amount > 0 {
		outputs = append(outputs, NewUtxo(accumulated-amount, from))
	}

	tx := &Transaction{
		Vin:  inputs,
		Vout: outputs,
	}

	tx.ID = tx.Hash()
	bc.SignTransaction(tx, fromWallet.PrivateKey)
	return tx
}

func (bc *Blockchain) MineBlock(txs []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	for _, tx := range txs {
		if !bc.VerifyTransaction(tx) {
			log.Panicf("交易%x验证失败", tx.ID)
		}
	}

	bc.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blockBucket))
		lastHash = bucket.Get([]byte("1"))

		blockData := bucket.Get(lastHash)
		lastBlock := Deserialize(blockData)
		lastHeight = lastBlock.Height

		return nil
	})

	block := NewBlock(txs, lastHash, lastHeight+1)

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

	return block
}

// 查找一笔交易
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("找不到这笔交易")
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privaKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			continue
		}
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(privaKey, prevTxs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTxs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			continue
		}
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTxs)
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
