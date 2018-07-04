package main

import (
	"fmt"
	"net"
	"log"
	"bytes"
	"encoding/gob"
)

const (
	protocol = "tcp"
	nodeVersion = 1
	commandLength = 12 // 用于指定命令
)

type Version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

var nodeAddress string
var minerAddress string
var knownNodes []string{"localhost:3000"}

func StartSever(nodeID, minerAddr string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	minerAddress = minerAddr

	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panicln(err)
	}
	defer ln.Close()

	bc := NewBlockchain(nodeID)

	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go handlerConn(conn, bc)
	}
}

func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	playload := gobEncode(Version{nodeVersion, bestHeight, nodeAddress})

	req := append(commandToBytes("version"), playload...)
	sendData(addr, req)
}

func sendData(addr string, req []byte) {
	conn, err := net.Dial()
}

func gobEncode(d interface{}) []byte {
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(&d)

	if err != nil {
		log.Panicln(err)
	}

	return buff.Bytes()
}

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func bytesToCommand(bs []byte) string {
	var command []byte

	for _, b := range bs {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}