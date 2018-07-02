package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)
const walletfile = "wallet.db"
const addressChecksumLen = 4

// 钱包不过就是一个私钥和公钥组成的秘钥对
type Wallet struct {
	PublicKey  []byte
	PrivateKey ecdsa.PrivateKey
}

// 创建一个钱包
func NewWallet() *Wallet {
	privateKey, pubKey := newKeyPair()
	wallet := &Wallet{
		PrivateKey: privateKey,
		PublicKey:  pubKey,
	}

	return wallet
}

func (w *Wallet) GetAddress() []byte {
	pubKeyHash := HashPubkey(w.PublicKey)
	versionedPlayload := append([]byte{version}, pubKeyHash...)

	checksum := checksum(versionedPlayload)
	full := append(versionedPlayload, checksum...)

	address := Base58Encode(full)

	return address
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panicln(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}

// 双重hash
func HashPubkey(pubkey []byte) []byte {
	publishSha256 := sha256.Sum256(pubkey)
	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publishSha256[:])

	if err != nil {
		log.Panicln(err)
	}

	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

func checksum(playload []byte) []byte {
	firstSha := sha256.Sum256(playload)
	secondSha := sha256.Sum256(firstSha[:])

	return secondSha[:addressChecksumLen]
}
