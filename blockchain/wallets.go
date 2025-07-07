package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
)

const walletFile = "wallet_%s.dat"

// Wallets stores a collection of wallets
type Wallets struct {
	Wallets map[string]*Wallet
}

// NewWallets creates Wallets and fills it from a file if it exists
func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile(nodeID)

	return &wallets, err
}

// CreateWallet adds a Wallet to Wallets
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = wallet

	return address
}

// GetAddresses returns an array of addresses stored in the wallet file
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// GetWallet returns a Wallet by its address
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// LoadFromFile loads wallets from the file
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var serializableWallets map[string]struct {
		PrivateKeyD     []byte
		PrivateKeyX     []byte
		PrivateKeyY     []byte
		PublicKey       []byte
	}
	
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&serializableWallets)
	
	if err != nil {
		// If we can't decode, the file might be in the old format or corrupted
		fmt.Printf("Warning: Could not decode wallet file: %v\n", err)
		// Delete the corrupted wallet file
		err = os.Remove(walletFile)
		if err != nil {
			log.Panic("Failed to delete corrupted wallet file: ", err)
		}
		return nil
	}
	
	// Successfully decoded, now convert back to wallet objects
	wallets := make(map[string]*Wallet)
	curve := elliptic.P256()
	
	for address, sw := range serializableWallets {
		privKey := &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: curve,
				X:     new(big.Int).SetBytes(sw.PrivateKeyX),
				Y:     new(big.Int).SetBytes(sw.PrivateKeyY),
			},
			D: new(big.Int).SetBytes(sw.PrivateKeyD),
		}
		
		wallets[address] = &Wallet{
			PrivateKey: *privKey,  // Notice we're dereferencing the pointer here to match Wallet struct
			PublicKey:  sw.PublicKey,
		}
	}
	
	ws.Wallets = wallets
	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveToFile(nodeID string) {
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeID)

	// Create a serializable version of the wallets
	type SerializableWallet struct {
		PrivateKeyD     []byte
		PrivateKeyX     []byte
		PrivateKeyY     []byte
		PublicKey       []byte
	}
	
	serializableWallets := make(map[string]SerializableWallet)
	
	for address, wallet := range ws.Wallets {
		serializableWallets[address] = SerializableWallet{
			PrivateKeyD:     wallet.PrivateKey.D.Bytes(),
			PrivateKeyX:     wallet.PrivateKey.PublicKey.X.Bytes(),
			PrivateKeyY:     wallet.PrivateKey.PublicKey.Y.Bytes(),
			PublicKey:       wallet.PublicKey,
		}
	}

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(serializableWallets)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
