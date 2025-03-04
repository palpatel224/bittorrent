package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	// "math"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
    Announce string `bencode:"announce"`
    Info     struct {
        Name        string `bencode:"name"`
        PieceLength int    `bencode:"piece length"`
        Pieces      string `bencode:"pieces"`
        Length      int    `bencode:"length,omitempty"`
        Files       []struct {
            Length int      `bencode:"length"`
            Path   []string `bencode:"path"`
        } `bencode:"files,omitempty"`
    } `bencode:"info"`
}

type TrackerResponse struct {
    FailureReason string `bencode:"failure reason"`
    Interval      int    `bencode:"interval"`
    Peers         string `bencode:"peers"`
}

func createHandshake(infoHash, peerID string) []byte {
    pstrlen := byte(19)
    pstr := "BitTorrent protocol"
    reserved := make([]byte, 8)
    infoHashBytes, _ := hex.DecodeString(infoHash)
    peerIDBytes := []byte(peerID)

    buf := new(bytes.Buffer)
    buf.WriteByte(pstrlen)
    buf.WriteString(pstr)
    buf.Write(reserved)
    buf.Write(infoHashBytes)
    buf.Write(peerIDBytes)

    return buf.Bytes()
}

func sendInterested(conn net.Conn) error {
    interested := []byte{0, 0, 0, 1, 2} // Length prefix (4 bytes) + message ID (1 byte)
    _, err := conn.Write(interested)
    return err
}

func waitForUnchoke(conn net.Conn) error {
    for {
        lengthBuf := make([]byte, 4)
        _, err := io.ReadFull(conn, lengthBuf)
        if err != nil {
            return err
        }

        length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
        if length == 0 {
            continue // Keep-alive message
        }

        messageBuf := make([]byte, length)
        _, err = io.ReadFull(conn, messageBuf)
        if err != nil {
            return err
        }

        messageID := messageBuf[0]
        if messageID == 1 { // Unchoke message
            return nil
        }
    }
}

func sendHave(conn net.Conn, pieceIndex int) error {
    have := make([]byte, 9)
    have[0] = 0
    have[1] = 0
    have[2] = 0
    have[3] = 5 // Length prefix (4 bytes)
    have[4] = 4 // Message ID (1 byte)
    have[5] = byte(pieceIndex >> 24)
    have[6] = byte(pieceIndex >> 16)
    have[7] = byte(pieceIndex >> 8)
    have[8] = byte(pieceIndex)
    _, err := conn.Write(have)
    return err
}

func requestPiece(conn net.Conn, index, begin, length int) error {
    request := make([]byte, 17)
    request[0] = 0
    request[1] = 0
    request[2] = 0
    request[3] = 13 // Length prefix (4 bytes)
    request[4] = 6  // Message ID (1 byte)
    request[5] = byte(index >> 24)
    request[6] = byte(index >> 16)
    request[7] = byte(index >> 8)
    request[8] = byte(index)
    request[9] = byte(begin >> 24)
    request[10] = byte(begin >> 16)
    request[11] = byte(begin >> 8)
    request[12] = byte(begin)
    request[13] = byte(length >> 24)
    request[14] = byte(length >> 16)
    request[15] = byte(length >> 8)
    request[16] = byte(length)
    _, err := conn.Write(request)
    return err
}

func receivePiece(conn net.Conn, expectedLength int) ([]byte, error) {
    header := make([]byte, 4+1+4+4) // Message length, ID, index, and begin
    _, err := io.ReadFull(conn, header)
    if err != nil {
        return nil, fmt.Errorf("error reading piece header: %w", err)
    }

    // Extract message ID and ensure it is a "Piece" message (ID = 7)
    if header[4] != 7 {
        return nil, fmt.Errorf("unexpected message ID: %d", header[4])
    }

    // Read the piece data
    piece := make([]byte, expectedLength)
    _, err = io.ReadFull(conn, piece)
    if err != nil {
        return nil, fmt.Errorf("error reading piece data: %w", err)
    }
    fmt.Printf("Received piece length: %d, expected length: %d\n", len(piece), expectedLength)
    return piece, nil
}

// func validatePiece(piece []byte, expectedHash []byte) bool {
//     hash := sha1.Sum(piece)
//     fmt.Printf("Expected hash: %x\n", expectedHash)
//     fmt.Printf("Actual hash: %x\n", hash[:])
//     return bytes.Equal(hash[:], expectedHash)
// }

// Define a struct to hold piece download results
type pieceResult struct {
    index int
    data  []byte
    err   error
}

func handlePeerConnection(address, infoHash, peerID string, torrent TorrentFile, resultChan chan<- pieceResult, pieceQueue <-chan int) {
    conn, err := net.DialTimeout("tcp", address, 5*time.Second)
    if err != nil {
        fmt.Printf("Error connecting to peer %s: %v\n", address, err)
        return
    }
    defer conn.Close()

    handshake := createHandshake(infoHash, peerID)
    _, err = conn.Write(handshake)
    if err != nil {
        fmt.Printf("Error sending handshake to peer %s: %v\n", address, err)
        return
    }

    response := make([]byte, 68)
    _, err = io.ReadFull(conn, response)
    if err != nil {
        fmt.Printf("Error reading handshake response from peer %s: %v\n", address, err)
        return
    }

    fmt.Printf("Received handshake response from peer %s\n", address)

    // Send Interested message
    err = sendInterested(conn)
    if err != nil {
        fmt.Printf("Error sending Interested message to peer %s: %v\n", address, err)
        return
    }

    // Wait for Unchoke message
    err = waitForUnchoke(conn)
    if err != nil {
        fmt.Printf("Error waiting for Unchoke message from peer %s: %v\n", address, err)
        return
    }

    fmt.Printf("Peer %s unchoked us\n", address)

    // Download pieces assigned to this connection
    for pieceIndex := range pieceQueue {
        pieceLength := torrent.Info.PieceLength
        fileLength := torrent.Info.Length
        numPieces := (fileLength + pieceLength - 1) / pieceLength
        blockSize := 1 << 14 // 16 KB

        var pieceBuffer []byte // Buffer to store concatenated blocks for this piece
        
        currentPieceLength := pieceLength
        if pieceIndex == numPieces-1 { // Last piece
            currentPieceLength = torrent.Info.Length % pieceLength
            if currentPieceLength == 0 { // If the file is a perfect multiple of pieceLength
                currentPieceLength = pieceLength
            }
        }
        fmt.Printf("[Peer %s] Downloading piece %d, length %d\n", address, pieceIndex, currentPieceLength)
        
        for begin := 0; begin < currentPieceLength; begin += blockSize {
            length := blockSize
            if begin+length > currentPieceLength {
                length = currentPieceLength - begin // Handle last block
            }
    
            // Request the block from the peer
            err = requestPiece(conn, pieceIndex, begin, length)
            if err != nil {
                fmt.Printf("[Peer %s] Error requesting block at piece %d, offset %d: %v\n", address, pieceIndex, begin, err)
                resultChan <- pieceResult{index: pieceIndex, data: nil, err: err}
                return
            }
    
            // Receive the block from the peer
            block, err := receivePiece(conn, length)
            if err != nil {
                fmt.Printf("[Peer %s] Error receiving block at piece %d, offset %d: %v\n", address, pieceIndex, begin, err)
                resultChan <- pieceResult{index: pieceIndex, data: nil, err: err}
                return
            }
    
            // Append the block to the piece buffer
            pieceBuffer = append(pieceBuffer, block...)
        }
    
        // All blocks for the piece received, calculate SHA-1 hash
        computedHash := sha1.Sum(pieceBuffer)
        expectedHash := []byte(torrent.Info.Pieces[pieceIndex*20 : (pieceIndex+1)*20])
    
        // Validate the piece
        if bytes.Equal(computedHash[:], expectedHash) {
            fmt.Printf("[Peer %s] Piece %d validated successfully\n", address, pieceIndex)
            resultChan <- pieceResult{index: pieceIndex, data: pieceBuffer, err: nil}
            err = sendHave(conn, pieceIndex)
            if err != nil {
                fmt.Printf("[Peer %s] Error sending Have message for piece %d: %v\n", address, pieceIndex, err)
            }
        } else {
            fmt.Printf("[Peer %s] Piece %d validation failed\n", address, pieceIndex)
            resultChan <- pieceResult{index: pieceIndex, data: nil, err: fmt.Errorf("piece validation failed")}
        }
    }
}

// Download torrent using multiple peers in parallel
func downloadTorrent(torrent TorrentFile, infoHashHex string, peerID string, peers []string) error {
    pieceLength := torrent.Info.PieceLength
    fileLength := torrent.Info.Length
    numPieces := (fileLength + pieceLength - 1) / pieceLength
    
    // Create channels for work distribution and result collection
    resultChan := make(chan pieceResult)
    pieceQueues := make([]chan int, len(peers))
    
    // Create a slice to store the pieces
    pieces := make([][]byte, numPieces)
    
    // Create a map to track which pieces are being downloaded
    inProgress := make(map[int]bool)
    
    // Create a slice to track which pieces need to be downloaded
    var pendingPieces []int
    for i := 0; i < numPieces; i++ {
        pendingPieces = append(pendingPieces, i)
    }
    
    // Start a goroutine for each peer
    for i, peer := range peers {
        pieceQueues[i] = make(chan int, 5) // Buffer for 5 pieces
        go handlePeerConnection(peer, infoHashHex, peerID, torrent, resultChan, pieceQueues[i])
    }
    
    // Start a goroutine to distribute work
    go func() {
        for len(pendingPieces) > 0 || len(inProgress) > 0 {
            // Assign pending pieces to available peers
            for _, queue := range pieceQueues {
                if len(pendingPieces) == 0 {
                    break
                }
                
                select {
                case queue <- pendingPieces[0]:
                    inProgress[pendingPieces[0]] = true
                    pendingPieces = pendingPieces[1:]
                default:
                    // Queue is full, try next peer
                    continue
                }
            }
            
            // Wait for results
            result := <-resultChan
            delete(inProgress, result.index)
            
            if result.err != nil {
                // If a piece failed, put it back in the pending list
                pendingPieces = append(pendingPieces, result.index)
                fmt.Printf("Piece %d failed, re-queuing\n", result.index)
            } else {
                // Store the successful piece
                pieces[result.index] = result.data
                fmt.Printf("Piece %d downloaded successfully (%d/%d)\n", 
                    result.index, numPieces-len(pendingPieces)-len(inProgress), numPieces)
            }
        }
        
        // Close all piece queues when done
        for _, queue := range pieceQueues {
            close(queue)
        }
    }()
    
    // Wait for all pieces to be downloaded
    for i := 0; i < numPieces; i++ {
        if pieces[i] == nil {
            i--
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    // Write the pieces to a file
    outputFile, err := os.Create(torrent.Info.Name)
    if err != nil {
        return fmt.Errorf("error creating output file: %v", err)
    }
    defer outputFile.Close()
    
    for _, piece := range pieces {
        if piece != nil {
            _, err = outputFile.Write(piece)
            if err != nil {
                return fmt.Errorf("error writing piece to file: %v", err)
            }
        }
    }
    
    fmt.Printf("File %s downloaded successfully\n", torrent.Info.Name)
    return nil
}

func main() {
    if len(os.Args) > 1 && os.Args[1] == "--cli" {
        // CLI mode
        if len(os.Args) < 3 {
            fmt.Println("Usage: main --cli <path to .torrent file>")
            return
        }
        
        filePath := os.Args[2]
        runCLI(filePath)
    } else {
        // GUI mode
        LaunchGUI()
    }
}

func runCLI(filePath string) {
    // Original CLI code
    file, err := os.Open(filePath)
    if err != nil {
        fmt.Printf("Error opening file: %v\n", err)
        return
    }
    defer file.Close()

    var torrent TorrentFile
    err = bencode.Unmarshal(file, &torrent)
    if err != nil {
        fmt.Printf("Error unmarshalling file: %v\n", err)
        return
    }

    fmt.Print("\n")
    fmt.Print("Torrent Info: ", torrent.Info, "\n")
    fmt.Print("\n\n")
    fmt.Print("Torrent Announce: ", torrent.Announce, "\n")
    fmt.Print("Torrent piecelength: ",torrent.Info.PieceLength, "\n")
    fmt.Print("Torrent length: ",torrent.Info.Length, "\n")
    fmt.Print("Torrent Announce: ", torrent.Info.Pieces, "\n")
    fmt.Print("\n\n")

    // Generate info_hash
    infoHash := sha1.New()
    err = bencode.Marshal(infoHash, torrent.Info)
    if err != nil {
        fmt.Printf("Error generating info_hash: %v\n", err)
        return
    }
    infoHashSum := infoHash.Sum(nil)
    infoHashHex := hex.EncodeToString(infoHashSum)
    fmt.Print("Info Hash: ", infoHashHex, "\n")


    // Construct the tracker URL
    params := url.Values{
        "info_hash":  {string(infoHashSum)},
        "peer_id":    {"-PC0001-123456789012"},
        "port":       {"6881"},
        "uploaded":   {"0"},
        "downloaded": {"0"},
        "left":       {fmt.Sprintf("%d", torrent.Info.Length)},
        "compact":    {"1"},
    }
    trackerURL := fmt.Sprintf("%s?%s", torrent.Announce, params.Encode())

    // Send GET request to the tracker
    resp, err := http.Get(trackerURL)
    if err != nil {
        fmt.Printf("Error sending GET request: %v\n", err)
        return
    }
    defer resp.Body.Close()

    var trackerResp TrackerResponse
    err = bencode.Unmarshal(resp.Body, &trackerResp)
    if err != nil {
        fmt.Printf("Error unmarshalling tracker response: %v\n", err)
        return
    }

    if trackerResp.FailureReason != "" {
        fmt.Printf("Tracker error: %s\n", trackerResp.FailureReason)
        return
    }

    fmt.Printf("Tracker response interval: %d seconds\n", trackerResp.Interval)

    // Process peers
    var peerAddresses []string
    peers := []byte(trackerResp.Peers)
    for i := 0; i < len(peers); i += 6 {
        ip := fmt.Sprintf("%d.%d.%d.%d", peers[i], peers[i+1], peers[i+2], peers[i+3])
        port := int(peers[i+4])<<8 + int(peers[i+5])
        address := fmt.Sprintf("%s:%d", ip, port)
        fmt.Printf("Peer: %s\n", address)
        peerAddresses = append(peerAddresses, address)
    }

    // Download the torrent using multiple peers in parallel
    err = downloadTorrent(torrent, infoHashHex, "-PC0001-123456789012", peerAddresses)
    if err != nil {
        fmt.Printf("Error downloading torrent: %v\n", err)
        return
    }
}