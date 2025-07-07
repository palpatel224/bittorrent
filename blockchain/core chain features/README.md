# Blockchain-Based BitTorrent Client

This project combines a cryptocurrency blockchain with a BitTorrent client to create a pay-to-access download system. Users must pay coins to a genesis address before they can use the BitTorrent download functionality.

## Features

- Complete blockchain implementation with wallets, transactions, and mining
- BitTorrent client capable of downloading from trackers and peers
- Payment verification system to protect access to the BitTorrent functionality
- Both CLI and GUI interfaces

## System Requirements

- Go 1.15 or higher
- For GUI: Additional dependencies for Fyne UI library

### GUI Dependencies

```sh
# Ubuntu/Debian
sudo apt-get install gcc libgl1-mesa-dev xorg-dev

# Fedora
sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
```

## Installation

1. **Clone the repository**:
   ```sh
   git clone <repository-url>
   cd blockchain-torrent
   ```

2. **Install dependencies**:
   ```sh
   go mod tidy
   ```

3. **Build the application**:
   ```sh
   go build .
   ```

## Usage

### GUI Mode

To launch the graphical user interface:

```sh
./blockchain-torrent gui
```

The GUI provides access to all functionality through an intuitive interface:

1. **Blockchain Operations**:
   - Create Blockchain
   - Create Wallet
   - List Addresses
   - Check Balance
   - Send Coins

2. **BitTorrent Downloads**:
   - Download Torrent (requires payment verification)

3. **Other Operations**:
   - Print Blockchain

### CLI Mode

For command-line interaction, set the NODE_ID environment variable (or it defaults to "3000"):

```sh
export NODE_ID=3000
```

Available commands:

1. **Create a blockchain with genesis block**:
   ```sh
   ./blockchain-torrent createblockchain -address ADDRESS
   ```

2. **Create a wallet**:
   ```sh
   ./blockchain-torrent createwallet
   ```

3. **List wallet addresses**:
   ```sh
   ./blockchain-torrent listaddresses
   ```

4. **Get balance for an address**:
   ```sh
   ./blockchain-torrent getbalance -address ADDRESS
   ```

5. **Send coins from one address to another**:
   ```sh
   ./blockchain-torrent send -from FROM -to TO -amount AMOUNT -mine
   ```
   The `-mine` flag mines the transaction on the same node.

6. **Print the blockchain**:
   ```sh
   ./blockchain-torrent printchain
   ```

7. **Access BitTorrent function (requires payment)**:
   ```sh
   ./blockchain-torrent accessfunction -address ADDRESS
   ```

8. **Download a torrent file (requires payment)**:
   ```sh
   ./blockchain-torrent downloadtorrent -from ADDRESS
   ```

## How the Payment System Works

1. When the blockchain is created, a genesis block is generated with a reward that goes to a specific address (the genesis address).

2. To access the BitTorrent download functionality, users must send at least 5 coins to this genesis address.

3. When attempting to download a torrent, the system verifies that the requesting address has made the required payment.

4. If payment verification succeeds, the BitTorrent client functionality becomes available and the user can download torrents.

## BitTorrent Client Features

- Supports .torrent file parsing
- Connects to trackers to get peer lists
- Downloads files using the BitTorrent protocol
- Validates downloaded pieces using SHA-1 hashing
- Supports multi-peer parallel downloads for efficiency

## Development

- The blockchain implementation is based on a simplified version of Bitcoin
- The BitTorrent client implements the core BitTorrent protocol
- GUI is built using the Fyne UI toolkit for Go


