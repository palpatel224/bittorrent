# Bittorrent in GO

This project is a BitTorrent client implemented in Go using the Bittorrent protocol. The client is capable of connecting to peers, requesting pieces of a file, and validating the received pieces using SHA-1 hashes. 
The main functionality is contained within the main.go file, which handles the connection to peers, sending requests for pieces, and receiving and validating the pieces.

## How to use
```bash
go run src/main.go <your-torrent-file>
```
Example:
You can clone the given repository and use the sample.torrent file
```bash
git clone https://github.com/vg239/bittorrent.git
```

```bash
go run src/main.go sample.torrent
```

The output of the file can be seen in the sample.txt

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

