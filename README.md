# Bittorrent in GO

This project is a BitTorrent client implemented in Go using the Bittorrent protocol. The client is capable of connecting to peers, requesting pieces of a file, and validating the received pieces using SHA-1 hashes. 
The main functionality is contained within the main.go file, which handles the connection to peers, sending requests for pieces, and receiving and validating the pieces.


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

The output of the exmaple file can be seen in the sample.txt or in the respective file name.

## Working
1) The client starts by reading a .torrent file to extract the necessary metadata, including the info hash, piece length, and the list of peers. 
2) It then sends a handshake to each peer and waits for an unchoke message before requesting pieces. 
3) Each piece is requested in blocks, and the received data is validated against the expected SHA-1 hash. 
4) Validated pieces are stored and eventually written to an output file.

The main.go file includes functions for : 
- Creating and sending handshake messages
- Sending requests for pieces
- Receiving pieces
- Validating the received data

