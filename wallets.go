package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallets() (*Wallets, error) {
	wallets := &Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile()
	return wallets, err
}

// 创建钱包
func (w *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())
	w.Wallets[address] = wallet
	return address
}

// 读取钱包集合，并反序列化
func (w *Wallets) LoadFromFile() error {
	if _, err := os.Stat(walletfile); os.IsNotExist(err) {
		return err
	}

	content, err := ioutil.ReadFile(walletfile)
	if err != nil {
		return err
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))
	decoder.Decode(&wallets)

	w.Wallets = wallets.Wallets
	return nil
}

// 保存钱包集合
func (w *Wallets) SaveToFile() {
	var buff bytes.Buffer

	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(w)
	if err != nil {
		log.Panicln(err)
	}

	err = ioutil.WriteFile(walletfile, buff.Bytes(), 0600)
	if err != nil {
		log.Panicln(err)
	}
}
