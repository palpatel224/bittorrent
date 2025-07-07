package main

import (
	"fmt"
	// "log"
)

func (cli *CLI) getBalance(address, nodeID string) {
	// TEMPORARY FIX: Comment out or modify the validation
	// if !ValidateAddress(address) {
	// 	log.Panic("ERROR: Address is not valid")
	// }

	// Print a warning instead
	if !ValidateAddress(address) {
		fmt.Println("WARNING: Address validation failed, but continuing anyway")
	}

	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}
