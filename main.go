package main

func main() {
	bc := NewBlockchain()
	defer bc.db.Close()
	cli := CLI{blockchain: bc}
	cli.Run()
}
