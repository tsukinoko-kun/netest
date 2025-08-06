package networktest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"github.com/tsukinoko-kun/netest/internal/history"
	"sync"
	"sync/atomic"
	"time"
)

const (
	testDownloadURL = "https://speed.cloudflare.com/__down?bytes=104857600" // 100MB
	testUploadURL   = "https://speed.cloudflare.com/__up"
	testLatencyURL  = "https://speed.cloudflare.com/__down?bytes=1"

	downloadTestDuration = 10 * time.Second
	uploadTestDuration   = 10 * time.Second
	latencyTestCount     = 10
	packetLossTestCount  = 20
)

type TestResults struct {
	DownloadSpeed float64       // Mbps
	UploadSpeed   float64       // Mbps
	Latency       time.Duration // Average latency
	PacketLoss    float64       // Percentage
	Jitter        time.Duration // Latency variation
}

func Run() error {
	results := TestResults{}
	var errs []error

	// Test latency and packet loss
	latency, jitter, packetLoss, err := testLatency()
	if err != nil {
		errs = append(errs, fmt.Errorf("latency test failed: %w", err))
	} else {
		results.Latency = latency
		results.Jitter = jitter
		results.PacketLoss = packetLoss
		fmt.Printf("Latency: %v (jitter: %v), Packet loss: %.1f%%\n", latency, jitter, packetLoss)
	}

	// Test download speed
	downloadSpeed, err := testDownloadSpeed()
	if err != nil {
		errs = append(errs, fmt.Errorf("download test failed: %w", err))
	} else {
		results.DownloadSpeed = downloadSpeed
		fmt.Printf("Download speed: %.2f Mbps\n", downloadSpeed)
	}

	// Test upload speed
	uploadSpeed, err := testUploadSpeed()
	if err != nil {
		errs = append(errs, fmt.Errorf("upload test failed: %w", err))
	} else {
		results.UploadSpeed = uploadSpeed
		fmt.Printf("Upload speed: %.2f Mbps\n", uploadSpeed)
	}

	if err := history.Track(results); err != nil {
		errs = append(errs, fmt.Errorf("failed to track results: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func testLatency() (avgLatency, jitter time.Duration, packetLoss float64, err error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		// Disable keep-alive to measure connection establishment time
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	var latencies []time.Duration
	failedRequests := 0

	for range packetLossTestCount {
		start := time.Now()

		resp, err := client.Get(testLatencyURL)
		if err != nil {
			failedRequests++
			continue
		}

		// Read and discard the response
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		latency := time.Since(start)
		latencies = append(latencies, latency)

		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	if len(latencies) == 0 {
		return 0, 0, 100, fmt.Errorf("all latency tests failed")
	}

	// Calculate average latency
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	avgLatency = total / time.Duration(len(latencies))

	// Calculate jitter (standard deviation)
	var variance float64
	for _, l := range latencies {
		diff := float64(l-avgLatency) / float64(time.Millisecond)
		variance += diff * diff
	}
	variance /= float64(len(latencies))
	stdDev := math.Sqrt(variance)
	jitter = time.Duration(stdDev) * time.Millisecond

	// Calculate packet loss
	packetLoss = float64(failedRequests) / float64(packetLossTestCount) * 100

	return avgLatency, jitter, packetLoss, nil
}

func testDownloadSpeed() (float64, error) {
	client := &http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTestDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testDownloadURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create download request: %w", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	// Continuously read data until timeout
	buf := make([]byte, 32*1024) // 32KB buffer
	var totalBytes int64

	for {
		n, err := resp.Body.Read(buf)
		totalBytes += int64(n)

		if err != nil {
			if err == io.EOF || errors.Is(err, context.DeadlineExceeded) {
				break
			}
			return 0, fmt.Errorf("download read error: %w", err)
		}
	}

	duration := time.Since(start).Seconds()
	speedMbps := (float64(totalBytes) * 8) / (duration * 1000000) // Convert to Mbps

	return speedMbps, nil
}

// uploadReader provides continuous data for upload testing
type uploadReader struct {
	data      []byte
	totalRead int64
	ctx       context.Context
}

func (r *uploadReader) Read(p []byte) (n int, err error) {
	// Check if context is done
	select {
	case <-r.ctx.Done():
		return 0, io.EOF
	default:
	}

	// Fill the buffer with our test data, repeating pattern as needed
	remaining := len(p)
	for remaining > 0 {
		toCopy := min(remaining, len(r.data))
		copied := copy(p[n:], r.data[:toCopy])
		n += copied
		remaining -= copied
	}

	atomic.AddInt64(&r.totalRead, int64(n))
	return n, nil
}

func testUploadSpeed() (float64, error) {
	client := &http.Client{
		Timeout: uploadTestDuration + 5*time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), uploadTestDuration)
	defer cancel()

	// Create test data pattern
	testPattern := bytes.Repeat([]byte("0123456789"), 1024) // 10KB pattern

	// Track total bytes uploaded across all goroutines
	var totalBytes atomic.Int64
	var wg sync.WaitGroup
	errChan := make(chan error, 5)

	start := time.Now()

	// Start multiple upload streams
	for i := range 3 {
		wg.Add(1)
		go func(streamID int) {
			defer wg.Done()

			reader := &uploadReader{
				data: testPattern,
				ctx:  ctx,
			}

			req, err := http.NewRequestWithContext(
				ctx,
				"POST",
				testUploadURL,
				reader,
			)
			if err != nil {
				errChan <- fmt.Errorf("stream %d: failed to create request: %w", streamID, err)
				return
			}

			// Set a large content length to allow continuous upload
			req.ContentLength = 1024 * 1024 * 1024 // 1GB (we won't actually upload this much)
			req.Header.Set("Content-Type", "application/octet-stream")

			resp, err := client.Do(req)
			if err != nil {
				// Context cancellation is expected
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					errChan <- fmt.Errorf("stream %d: upload failed: %w", streamID, err)
				}
				// Add the bytes uploaded before error
				totalBytes.Add(atomic.LoadInt64(&reader.totalRead))
				return
			}
			defer resp.Body.Close()

			// Read response
			_, _ = io.Copy(io.Discard, resp.Body)

			// Add bytes from this stream
			totalBytes.Add(atomic.LoadInt64(&reader.totalRead))
		}(i)
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any errors
	var uploadErr error
	for err := range errChan {
		if uploadErr == nil {
			uploadErr = err
		}
	}

	duration := time.Since(start).Seconds()
	if duration == 0 {
		return 0, fmt.Errorf("upload test completed too quickly")
	}

	bytesUploaded := totalBytes.Load()
	if bytesUploaded == 0 {
		return 0, fmt.Errorf("no data was uploaded")
	}

	speedMbps := (float64(bytesUploaded) * 8) / (duration * 1000000) // Convert to Mbps

	if uploadErr != nil {
		return speedMbps, uploadErr
	}

	return speedMbps, nil
}
