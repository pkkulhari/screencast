package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kbinani/screenshot"
)

const (
	sleepTime     = 33 * time.Millisecond // ~30 FPS
	listenAddress = ":8080"
)

var (
	currentFrame     []byte
	currentFrameLock sync.RWMutex
	stopCapture      = make(chan struct{})
)

func captureScreen() {
	// Get primary display bounds
	bounds := screenshot.GetDisplayBounds(0)

	for {
		select {
		case <-stopCapture:
			return
		default:
			// Capture screenshot
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				log.Printf("Error capturing screen: %v", err)
				time.Sleep(sleepTime)
				continue
			}

			// Convert image to PNG
			buf := new(bytes.Buffer)
			err = png.Encode(buf, img)
			if err != nil {
				log.Printf("Error encoding PNG: %v", err)
				time.Sleep(sleepTime)
				continue
			}

			// Update current frame
			currentFrameLock.Lock()
			currentFrame = buf.Bytes()
			currentFrameLock.Unlock()

			time.Sleep(sleepTime)
		}
	}
}

func frameHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

	for {
		currentFrameLock.RLock()
		frame := currentFrame
		currentFrameLock.RUnlock()

		if frame == nil {
			time.Sleep(sleepTime)
			continue
		}

		// Write multipart frame
		fmt.Fprintf(w, "--frame\r\nContent-Type: image/png\r\n\r\n")
		w.Write(frame)
		w.Write([]byte("\r\n"))

		// Flush to send data immediately
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		time.Sleep(sleepTime)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Screen Stream</title>
	</head>
	<body>
		<div style="text-align: center;">
			<img src="/frame" alt="Screen Stream">
		</div>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	// Start screen capture goroutine
	go captureScreen()

	// Setup HTTP routes
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/frame", frameHandler)

	// Start server
	fmt.Printf("Screen stream server running on http://localhost%s\n", listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}
