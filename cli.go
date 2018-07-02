package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type CLI struct {
}

func (cli *CLI) printUsage() {
	fmt.Println("用法如下：")
	fmt.Println("\tcreateblockchain -address data - 创建一个新的区块链并把创世区块的奖励给矿工")
	fmt.Println("\tshowblockchain 展示区块链")
	fmt.Println("\tgetbalance -address data 获取某地址余额")
	fmt.Println("\tsend -from address -to address -amount money 从from地址支付amount到to地址")
	fmt.Println("\tcreatewallet 创建一个钱包")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) showBlockchain() {
	bc := NewBlockchain("")
	bci := bc.Iterator()
	for {
		b := bci.Next()
		fmt.Printf("上一块哈希:%x\n", b.PrevHash)
		fmt.Printf("当前哈希:%x\n", b.Hash)
		pow := NewProofOfWork(b)
		fmt.Println("pow is", pow.IsValidate())
		fmt.Println()

		if len(b.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CLI) CreateBlockchain(address string) {
	bc := Createblockchain(address)
	defer bc.CloseDb()
	fmt.Println("Done")
}

func (cli *CLI) GetBalance(address string) {
	bc := NewBlockchain(address)
	defer bc.CloseDb()

	var balance int
	utxo := bc.FindUTXO(address)

	for _, out := range utxo {
		balance += out.Value
	}

	fmt.Printf("address:%s's balance is %d\n", address, balance)
}

func (cli *CLI) Send(from string, to string, amount int) {
	bc := NewBlockchain("")
	defer bc.CloseDb()

	tx := bc.NewUTXOTransaction(from, to, amount)

	// 挖矿，把此笔交易附加到区块中
	bc.MineBlock([]*Transaction{tx})

	fmt.Println("Done!")
}

func (cli *CLI) CreateWallet() {
	ws, err := NewWallets()
	if err != nil {
		log.Panicln(err)
	}
	address := ws.CreateWallet()
	ws.SaveToFile()

	fmt.Printf("创建钱包成功，你的地址是%s\n", address)
}

func (cli *CLI) Run() {
	cli.validateArgs()

	showBlockchainCmd := flag.NewFlagSet("showblockchain", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)

	createBlockchainData := createBlockchainCmd.String("address", "", "init address")
	getBalanceData := getBalanceCmd.String("address", "", "查看余额的address")

	from := sendCmd.String("from", "", "发送地址")
	to := sendCmd.String("to", "", "接受地址")
	amount := sendCmd.Int("amount", 0, "发送币数")

	switch os.Args[1] {
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panicln(err)
		}
	case "showblockchain":
		err := showBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panicln(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panicln(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panicln(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if showBlockchainCmd.Parsed() {
		cli.showBlockchain()
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainData == "" {
			createBlockchainCmd.Usage()
		} else {
			cli.CreateBlockchain(*createBlockchainData)
		}
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceData == "" {
			getBalanceCmd.Usage()
		} else {
			cli.GetBalance(*getBalanceData)
		}
	}

	if sendCmd.Parsed() {
		if *from == "" || *to == "" || *amount == 0 {
			getBalanceCmd.Usage()
		} else {
			cli.Send(*from, *to, *amount)
		}
	}
}
