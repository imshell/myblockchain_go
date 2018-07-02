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
	WalletMap map[string]*Wallet
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.WalletMap = make(map[string]*Wallet)
	err := wallets.LoadFromFile()
	return &wallets, err
}

func (w *Wallets) GetWallet(address string) *Wallet {
	return w.WalletMap[address]
}

// 创建钱包
func (w *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	w.WalletMap[address] = wallet
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

	if len(content) == 0 {
		return nil
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))
	decoder.Decode(&wallets)

	w.WalletMap = wallets.WalletMap
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
