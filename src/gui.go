package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/jackpal/bencode-go"
)

type DownloadProgress struct {
	TotalPieces      int
	DownloadedPieces int
	ProgressBar      *widget.ProgressBar
	StatusLabel      *widget.Label
	Window           fyne.Window
}

func (dp *DownloadProgress) UpdateProgress(pieceIndex int, total int) {
	dp.DownloadedPieces++
	progress := float64(dp.DownloadedPieces) / float64(dp.TotalPieces)

	// Update UI from the main thread
	dp.Window.Canvas().Refresh(dp.ProgressBar)
	dp.ProgressBar.SetValue(progress)
	dp.StatusLabel.SetText(fmt.Sprintf("Downloading: %d/%d pieces (%.1f%%)",
		dp.DownloadedPieces, dp.TotalPieces, progress*100))
}

func (dp *DownloadProgress) SetComplete() {
	dp.ProgressBar.SetValue(1.0)
	dp.StatusLabel.SetText("Download complete!")
}

func (dp *DownloadProgress) SetError(err error) {
	dp.StatusLabel.SetText(fmt.Sprintf("Error: %v", err))
}

// Function to normalize file paths for different operating systems
func normalizePath(path string) string {
	// For Windows, convert forward slashes to backslashes for display
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(path, "/", "\\")
	}
	return path
}

// Create the main UI content
func createMainContent(w fyne.Window) fyne.CanvasObject {
	// File selection
	filePathEntry := widget.NewEntry()
	filePathEntry.SetPlaceHolder("Path to .torrent file")
	
	browseButton := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil || uri == nil {
				return
			}
			
			// Get the path and normalize it for display
			path := uri.URI().Path()
			
			// On some platforms, the path might have a leading slash that needs to be removed
			if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") {
				path = path[1:]
			}
			
			// Normalize and display the path
			filePathEntry.SetText(normalizePath(path))
		}, w)
	})
	
	// Download button
	downloadButton := widget.NewButton("Download", func() {
		filePath := filePathEntry.Text
		if filePath == "" {
			dialog.ShowError(fmt.Errorf("please select a .torrent file"), w)
			return
		}
		
		// Create progress tracking
		progressBar := widget.NewProgressBar()
		statusLabel := widget.NewLabel("Preparing download...")
		
		progress := &DownloadProgress{
			ProgressBar: progressBar,
			StatusLabel: statusLabel,
			Window:      w,
		}
		
		// Back button to return to main screen
		backButton := widget.NewButton("Back", func() {
			w.SetContent(createMainContent(w))
		})
		
		// Replace the content with the progress view
		w.SetContent(container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Downloading: %s", filepath.Base(filePath))),
			progressBar,
			statusLabel,
			backButton,
		))
		
		// Start the download in a goroutine
		go func() {
			err := downloadTorrentWithGUI(filePath, progress)
			if err != nil {
				progress.SetError(err)
			} else {
				progress.SetComplete()
			}
		}()
	})
	
	// Layout
	return container.NewVBox(
		widget.NewLabel("BitTorrent Client"),
		container.NewHBox(filePathEntry, browseButton),
		downloadButton,
	)
}

func LaunchGUI() {
	a := app.New()
	w := a.NewWindow("BitTorrent Client")
	w.Resize(fyne.NewSize(600, 400))

	// Set the initial content
	w.SetContent(createMainContent(w))
	w.ShowAndRun()
}

func downloadTorrentWithGUI(filePath string, progress *DownloadProgress) error {
	// Handle Windows path format if needed
	if runtime.GOOS == "windows" && !strings.Contains(filePath, ":\\") {
		// Convert path format if it's not already in Windows format
		filePath = strings.ReplaceAll(filePath, "/", "\\")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var torrent TorrentFile
	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		return fmt.Errorf("error unmarshalling file: %v", err)
	}

	// Generate info_hash
	infoHash := sha1.New()
	err = bencode.Marshal(infoHash, torrent.Info)
	if err != nil {
		return fmt.Errorf("error generating info_hash: %v", err)
	}
	infoHashSum := infoHash.Sum(nil)
	infoHashHex := hex.EncodeToString(infoHashSum)

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
		return fmt.Errorf("error sending GET request: %v", err)
	}
	defer resp.Body.Close()

	var trackerResp TrackerResponse
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return fmt.Errorf("error unmarshalling tracker response: %v", err)
	}

	if trackerResp.FailureReason != "" {
		return fmt.Errorf("tracker error: %s", trackerResp.FailureReason)
	}

	// Process peers
	var peerAddresses []string
	peers := []byte(trackerResp.Peers)
	for i := 0; i < len(peers); i += 6 {
		ip := fmt.Sprintf("%d.%d.%d.%d", peers[i], peers[i+1], peers[i+2], peers[i+3])
		port := int(peers[i+4])<<8 + int(peers[i+5])
		address := fmt.Sprintf("%s:%d", ip, port)
		peerAddresses = append(peerAddresses, address)
	}

	// Set up progress tracking
	pieceLength := torrent.Info.PieceLength
	fileLength := torrent.Info.Length
	numPieces := (fileLength + pieceLength - 1) / pieceLength
	progress.TotalPieces = numPieces

	// Download the torrent using multiple peers in parallel with progress updates
	return downloadTorrentWithProgress(torrent, infoHashHex, "-PC0001-123456789012", peerAddresses, progress)
}

func downloadTorrentWithProgress(torrent TorrentFile, infoHashHex string, peerID string, peers []string, progress *DownloadProgress) error {
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
			} else {
				// Store the successful piece
				pieces[result.index] = result.data
				// Update progress in the GUI
				progress.UpdateProgress(result.index, numPieces)
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

	return nil
}
