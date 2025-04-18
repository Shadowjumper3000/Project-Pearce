package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-toast/toast"
)

const (
    // Alternative download test URLs
    downloadURL = "https://hil-speed.hetzner.com/100MB.bin" // 100MB file from Cloudflare
    // downloadURL = "http://speedtest.ftp.otenet.gr/files/test100Mb.db" // Alternative 100MB test file
    uploadURL   = "https://httpbin.org/post"                            // Endpoint for upload test
    testSize    = 10 << 20                        // 10 MB for upload test (adjustable)
)

func main() {
    // Run speed tests
    downloadSpeed, downloadErr := testDownloadSpeed()
    uploadSpeed, uploadErr := testUploadSpeed()
    
    // Format results message
    var resultMsg string
    if downloadErr != nil {
        resultMsg = fmt.Sprintf("❌ Download test failed: %v\n", downloadErr)
    } else {
        resultMsg = fmt.Sprintf("⬇️ Download Speed: %.2f Mbps\n", downloadSpeed)
    }
    
    if uploadErr != nil {
        resultMsg += fmt.Sprintf("❌ Upload test failed: %v", uploadErr)
    } else {
        resultMsg += fmt.Sprintf("⬆️ Upload Speed: %.2f Mbps", uploadSpeed)
    }
    
    // Create and show Windows notification
    notification := toast.Notification{
        AppID:   "SpeedTest",
        Title:   "Speed Test Results",
        Message: resultMsg,
    }
    
    err := notification.Push()
    if err != nil {
        // Create a file for logging errors since console isn't visible
        f, _ := os.OpenFile("speedtest_error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if f != nil {
            fmt.Fprintf(f, "Error showing notification: %v\n", err)
            fmt.Fprintln(f, resultMsg)
            f.Close()
        }
        
        // Still show console output as fallback (won't be visible in GUI mode but useful in dev)
        fmt.Println("Error showing notification:", err)
        fmt.Println("🚀 Internet Speed Test Results:")
        fmt.Println(resultMsg)
        fmt.Print("\nPress Enter to exit...")
        fmt.Scanln()
    }
}

// testDownloadSpeed measures download speed in Mbps
func testDownloadSpeed() (float64, error) {
    startTime := time.Now()
    
    // Create buffer to track actual bytes downloaded
    var byteCounter int64
    
    resp, err := http.Get(downloadURL)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    // Use a counting reader to accurately measure downloaded bytes
    countReader := &countingReader{reader: resp.Body, counter: &byteCounter}
    
    // Read the entire response (discard it but count bytes)
    _, err = io.Copy(io.Discard, countReader)
    if err != nil {
        return 0, err
    }

    // Use the actual bytes downloaded, not the Content-Length header
    duration := time.Since(startTime).Seconds()
    fileSize := float64(byteCounter) * 8 / 1_000_000 // Convert bytes to megabits
    
    // Avoid division by zero
    if duration <= 0 {
        return 0, fmt.Errorf("test completed too quickly to measure")
    }
    
    speed := fileSize / duration

    return speed, nil
}

// countingReader wraps a reader and counts bytes read
type countingReader struct {
    reader  io.Reader
    counter *int64
}

func (r *countingReader) Read(p []byte) (n int, err error) {
    n, err = r.reader.Read(p)
    *r.counter += int64(n)
    return
}

// testUploadSpeed measures upload speed in Mbps
func testUploadSpeed() (float64, error) {
    // Generate a 10MB dummy payload
    data := bytes.Repeat([]byte("0"), testSize)
    startTime := time.Now()

    resp, err := http.Post(uploadURL, "application/octet-stream", bytes.NewReader(data))
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    duration := time.Since(startTime).Seconds()
    fileSize := float64(testSize) * 8 / 1_000_000 // Convert bytes to megabits
    speed := fileSize / duration

    return speed, nil
}