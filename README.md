# Bittorrent in GO

This project is a BitTorrent client implemented in Go using the Bittorrent protocol. The client is capable of connecting to peers, requesting pieces of a file, and validating the received pieces using SHA-1 hashes. 
The main functionality is contained within the main.go file, which handles the connection to peers, sending requests for pieces, and receiving and validating the pieces.

A report post analyzing results using different network conditions, peer handling and piece sizes has been added here : <a href= "https://drive.google.com/file/d/1JrY8lA2z0Le2DXva6bTLqklNDD17I_kb/view">Report Link</a>

## How to Use

### GUI
**Make sure you have the latest version of Go installed along with some platform dependencies for gui**
```sh
sudo apt-get install gcc libgl1-mesa-dev xorg-dev
```
### BitTorrent Client

1. **Clone the repository**:
    ```sh
    git clone https://github.com/vg239/bittorrent.git
    cd bittorrent-client
    ```

2. **Install dependencies**:
    ```sh
    go mod tidy
    ```

3. **Build the application**:
    ```sh
    go build -o bittorrent-client src/*.go
    ```

4. **Use GUI**:
    ```sh
    ./bittorrent-client
    ```

5. **Use CLI**:
    ```sh
    ./bittorrent-client --cli <path-to-torrent-file>
    ```

The output of the example file can be seen in the sample.txt or in the respective file name.

## Working
1) The client starts by reading a .torrent file to extract the necessary metadata, including the info hash, piece length, and the list of peers. 
2) It then sends a handshake to each peer and waits for an unchoke message before requesting pieces. 
3) Each piece is requested in blocks, and the received data is validated against the expected SHA-1 hash. 
4) Validated pieces are stored and eventually written to an output file.

The main.go file includes functions for:
- Creating and sending handshake messages
- Sending requests for pieces
- Receiving pieces
- Validating the received data

## Blockchain Integration Details

The `/blockchain` directory contains a complete blockchain implementation that can be used to create a pay-to-access system for the BitTorrent client. Key features include:

- Full blockchain with wallets, transactions, and mining
- Payment verification system
- CLI interface for blockchain operations
- Minimum payment requirement of 5 coins to genesis address
- Automatic verification of payments before allowing downloads

For more details about the blockchain implementation, please refer to the README in the `/blockchain` directory.

## Optional Blockchain Integration

This client can optionally be integrated with a blockchain-based payment system located in the `/blockchain` directory. When enabled, access to the BitTorrent functionality is protected by a blockchain payment verification system.

### Blockchain Payment Setup (Optional)

If you wish to use the blockchain payment system:

1. **Set your node ID**:
    ```sh
    export NODE_ID=3000
    ```

2. **Create a wallet**:
    ```sh
    cd blockchain
    ./blockchain createwallet
    ```

3. **Create the blockchain** (if not already created):
    ```sh
    ./blockchain createblockchain -address YOUR_WALLET_ADDRESS
    ```

4. **Send payment to access BitTorrent**:
    ```sh
    ./blockchain send -from YOUR_WALLET_ADDRESS -to GENESIS_ADDRESS -amount 5 -mine
    ```

5. **Verify access**:
    ```sh
    ./blockchain accessfunction -address YOUR_WALLET_ADDRESS
    ```

Once payment is verified, you can proceed with the standard BitTorrent functionality.


