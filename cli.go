package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type CLI struct {
	blockchain *Blockchain
}

func (cli *CLI) printUsage() {
	fmt.Println("用法如下:")
	fmt.Println("addBlock 添加一个区块")
	fmt.Println("showBlockchain 展示区块链")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) showBlockchain() {
	bci := cli.blockchain.Iterator()
	for {
		block := bci.Next()
		fmt.Printf("上一块哈希:%x\n", block.PrevHash)
		fmt.Printf("数据:%s\n", block.Data)
		fmt.Printf("当前哈希:%x\n", block.Hash)
		pow := NewProofOfWork(block)
		fmt.Println("pow is", pow.IsValidate())

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CLI) addBlock(data string) {
	cli.blockchain.AddBlock(data)
}

func (cli *CLI) Run() {
	addBlockCmd := flag.NewFlagSet("addblock", flag.ExitOnError)
	showBlockchainCmd := flag.NewFlagSet("showblockchain", flag.ExitOnError)

	addBlockData := addBlockCmd.String("data", "", "Block Data")

	switch os.Args[1] {
	case "addblock":
		err := addBlockCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panicln(err)
		}
	case "showchain":
		err := showBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panicln(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
		} else {
			cli.addBlock(*addBlockData)
		}
	}

	if showBlockchainCmd.Parsed() {
		cli.showBlockchain()
	}
}
