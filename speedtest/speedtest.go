package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-toast/toast"
)

// CloudflareSpeedTest represents the response from Cloudflare's speed test API
type CloudflareSpeedTest struct {
	DownloadSpeed float64 `json:"download"`
	UploadSpeed   float64 `json:"upload"`
	Latency       float64 `json:"latency"`
	ISP           string  `json:"isp"`
	ServerName    string  `json:"server"`
}

func main() {
	// Wait for network connectivity
	if !waitForNetwork() {
		showToastNotification("Network connection not detected")
		return
	}

	var test CloudflareSpeedTest
	var err error
	
	// Try up to 3 times in case of API failure
	for i := 0; i < 3; i++ {
		test, err = runCloudflareSpeedTest()
		if err == nil {
			break
		}
		// Short pause before retry
		time.Sleep(2 * time.Second)
	}

	// Format results message
	var resultMsg string

	if err != nil {
		resultMsg = fmt.Sprintf("Speed test failed: %v", err)
	} else {
		resultMsg = fmt.Sprintf("Ping: %.0f ms (%s)\n", test.Latency, test.ServerName)
		resultMsg += fmt.Sprintf("↓ Download: %.2f Mbps\n", test.DownloadSpeed)
		resultMsg += fmt.Sprintf("↑ Upload: %.2f Mbps", test.UploadSpeed)
	}

	// Show results as notification
	showToastNotification(resultMsg)
}

// waitForNetwork checks for network connectivity
func waitForNetwork() bool {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			if checkNetworkConnectivity() {
				return true
			}
		}
	}
}

// checkNetworkConnectivity verifies if we have internet access
func checkNetworkConnectivity() bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Try connecting to a reliable endpoint
	_, err := client.Get("https://1.1.1.1")
	return err == nil
}

// runCloudflareSpeedTest runs a speed test using our own implementation
func runCloudflareSpeedTest() (CloudflareSpeedTest, error) {
	// Since the Cloudflare API is not working, let's implement our own tests
	
	// 1. Test latency
	latency, err := testLatency()
	if err != nil {
		latency = 0 // Continue even if ping fails
	}
	
	// 2. Test download speed
	downloadSpeed, err := testDownload()
	if err != nil {
		downloadSpeed = 0 // Continue even if download fails
	}
	
	// 3. Test upload speed
	uploadSpeed, err := testUpload()
	if err != nil {
		uploadSpeed = 0 // Continue even if upload fails
	}
	
	// Return combined results
	result := CloudflareSpeedTest{
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   uploadSpeed,
		Latency:       latency,
		ISP:           "Your ISP",
		ServerName:    "Cloudflare",
	}
	
	// Verify we have at least some data
	if downloadSpeed == 0 && uploadSpeed == 0 && latency == 0 {
		return result, fmt.Errorf("all speed tests failed")
	}
	
	return result, nil
}

// testLatency measures average ping time to multiple servers
func testLatency() (float64, error) {
	// Ping reliable servers
	targets := []string{
		"1.1.1.1",       // Cloudflare
		"8.8.8.8",       // Google
		"9.9.9.9",       // Quad9
	}
	
	var totalLatency float64
	var successCount int
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	for _, target := range targets {
		startTime := time.Now()
		
		resp, err := client.Get("https://" + target)
		if err != nil {
			continue // Try next target
		}
		defer resp.Body.Close()
		
		// Read a small amount to ensure the connection is established
		io.CopyN(io.Discard, resp.Body, 1024)
		
		pingTime := time.Since(startTime).Milliseconds()
		totalLatency += float64(pingTime)
		successCount++
	}
	
	if successCount == 0 {
		return 0, fmt.Errorf("failed to ping any servers")
	}
	
	return totalLatency / float64(successCount), nil
}

// testDownload measures download speed using multiple connections
func testDownload() (float64, error) {
	// Configuration
	const connections = 4           // Use multiple parallel connections like web speed tests
	const downloadDuration = 8      // Test for longer to get more accurate results
	
	// Download URL
	downloadURL := "https://speed.cloudflare.com/__down?bytes=25000000" // 25MB
	
	// Synchronization
	var wg sync.WaitGroup
	var totalBytesDownloaded int64
	
	// Start time for all connections
	startTime := time.Now()
	deadline := startTime.Add(time.Duration(downloadDuration) * time.Second)
	
	// Create a client with appropriate timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: false,
			MaxIdleConns: connections * 2,
			MaxIdleConnsPerHost: connections * 2,
		},
	}
	
	// Start multiple download threads
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Keep downloading until time is up
			for time.Now().Before(deadline) {
				resp, err := client.Get(downloadURL)
				if err != nil {
					continue // Try again
				}
				
				// Read and count bytes
				bytes, err := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				
				if err == nil {
					atomic.AddInt64(&totalBytesDownloaded, bytes)
				}
			}
		}()
	}
	
	// Wait for all downloads to complete
	wg.Wait()
	
	// Calculate results
	testDuration := time.Since(startTime).Seconds()
	if testDuration < 1 || totalBytesDownloaded == 0 {
		return 0, fmt.Errorf("insufficient data for measurement")
	}
	
	// Calculate speed in Mbps
	downloadSpeed := (float64(totalBytesDownloaded) * 8 / 1_000_000) / testDuration
	
	return downloadSpeed, nil
}

// testUpload measures upload speed with multiple connections
func testUpload() (float64, error) {
	// Configuration
	const connections = 3           // Use multiple connections
	const uploadDuration = 8        // Test for longer duration
	const chunkSize = 1 * 1024 * 1024  // 1MB chunks
	
	uploadURL := "https://httpbin.org/post" // Use httpbin for upload testing
	
	// Synchronization
	var wg sync.WaitGroup
	var totalBytesUploaded int64
	
	// Generate test data once (reused across connections)
	data := make([]byte, chunkSize)
	for i := range data {
		data[i] = byte(i % 256) // Add some variation to avoid compression effects
	}
	
	// Start time for all connections
	startTime := time.Now()
	deadline := startTime.Add(time.Duration(uploadDuration) * time.Second)
	
	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: false,
			MaxIdleConns: connections * 2,
			MaxIdleConnsPerHost: connections * 2,
		},
	}
	
	// Start multiple upload threads
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Keep uploading until time is up
			for time.Now().Before(deadline) {
				resp, err := client.Post(uploadURL, "application/octet-stream", bytes.NewReader(data))
				if err != nil {
					continue // Try again
				}
				
				// Discard the response body but make sure to read it all
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				
				// Count the uploaded bytes
				atomic.AddInt64(&totalBytesUploaded, int64(len(data)))
			}
		}()
	}
	
	// Wait for all uploads to complete
	wg.Wait()
	
	// Calculate results
	testDuration := time.Since(startTime).Seconds()
	if testDuration < 1 || totalBytesUploaded == 0 {
		return 0, fmt.Errorf("insufficient data for measurement")
	}
	
	// Calculate speed in Mbps
	uploadSpeed := (float64(totalBytesUploaded) * 8 / 1_000_000) / testDuration
	
	return uploadSpeed, nil
}

// infiniteZeroReader provides an infinite stream of zeros
type infiniteZeroReader struct{}

func (*infiniteZeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// showToastNotification shows a system notification with the results
func showToastNotification(message string) {
	// Different notification methods per OS
	switch runtime.GOOS {
	case "windows":
		showWindowsToast(message)
	case "darwin":
		showMacNotification(message)
	case "linux":
		showLinuxNotification(message)
	default:
		fmt.Println("\n--- Speed Test Results ---")
		fmt.Println(message)
	}
}

// showWindowsToast shows a notification on Windows
func showWindowsToast(message string) {
	// List of AppIDs to try in order
	appIDs := []string{
		"SpeedTest",
		"Microsoft.Windows.Shell.RunDialog",
		"Windows.SystemToast.ShellFeedHost.Notification",
		"Microsoft.WindowsStore_8wekyb3d8bbwe!App",
		"Microsoft.Windows.Explorer",
		"{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\\WindowsPowerShell\\v1.0\\powershell.exe",
		"Windows.SystemToast.Winlogon.Notification",
	}

	notification := toast.Notification{
		AppID:   "SpeedTest",
		Title:   "Speed Test Results",
		Message: message,
		Icon:    "",
		Audio:   toast.Silent,
		Actions: []toast.Action{
			{Type: "protocol", Label: "Close", Arguments: ""},
		},
		Duration: toast.Long,
	}

	// Try each AppID until one works
	var lastErr error
	for _, appID := range appIDs {
		notification.AppID = appID
		err := notification.Push()
		if err == nil {
			return
		}
		lastErr = err
	}

	// Fallback to console if all notifications fail
	fmt.Println("\n--- Speed Test Results ---")
	fmt.Println(message)
	if lastErr != nil {
		fmt.Printf("(Notification failed: %v)\n", lastErr)
	}
}

// showMacNotification shows a notification on macOS
func showMacNotification(message string) {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`display notification "%s" with title "Speed Test Results"`, message))
	if err := cmd.Run(); err != nil {
		fmt.Println("\n--- Speed Test Results ---")
		fmt.Println(message)
		fmt.Printf("(Notification failed: %v)\n", err)
	}
}

// showLinuxNotification shows a notification on Linux
func showLinuxNotification(message string) {
	// Try notify-send first
	if _, err := exec.LookPath("notify-send"); err == nil {
		cmd := exec.Command("notify-send", "Speed Test Results", message)
		if err := cmd.Run(); err != nil {
			fmt.Println("\n--- Speed Test Results ---")
			fmt.Println(message)
			fmt.Printf("(Notification failed: %v)\n", err)
		}
		return
	}

	// Fallback to console
	fmt.Println("\n--- Speed Test Results ---")
	fmt.Println(message)
}