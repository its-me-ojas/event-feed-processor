package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	baseURL     string
	concurrency int
	requests    int
	mode        string
	client      *http.Client
)

func init() {
	flag.StringVar(&baseURL, "url", "http://localhost:8080", "Base URL of the API")
	flag.IntVar(&concurrency, "c", 10, "Number of concurrent workers")
	flag.IntVar(&requests, "n", 1000, "Total number of requests to send")
	flag.StringVar(&mode, "mode", "write", "Mode: 'write' (create posts) or 'read' (get feeds)")
	flag.Parse()

	// Initialize global client with pooled connections
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	client = &http.Client{
		Timeout:   10 * time.Second,
		Transport: t,
	}
}

func main() {
	fmt.Printf("🚀 Starting Load Test (%s)\n", mode)
	fmt.Printf("   URL: %s\n   Concurrency: %d\n   Total Requests: %d\n\n", baseURL, concurrency, requests)

	start := time.Now()
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	var totalLatency int64

	jobs := make(chan int, requests)

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				startReq := time.Now()
				err := makeRequest()
				latency := time.Since(startReq).Microseconds()
				atomic.AddInt64(&totalLatency, latency)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					// fmt.Println("Error:", err) // Optional: uncomment for verbose error logs
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	// Queue jobs
	for i := 0; i < requests; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	duration := time.Since(start)

	// Report
	rps := float64(successCount) / duration.Seconds()
	var avgLatency float64
	if successCount > 0 {
		avgLatency = float64(atomic.LoadInt64(&totalLatency)) / float64(successCount) / 1000.0 // ms
	}

	fmt.Println("📊 Results:")
	fmt.Printf("   Time Taken: %v\n", duration)
	fmt.Printf("   Requests: %d\n", requests)
	fmt.Printf("   Success: %d\n", successCount)
	fmt.Printf("   Errors: %d\n", errorCount)
	fmt.Printf("   RPS: %.2f req/s\n", rps)
	fmt.Printf("   Avg Latency: %.2f ms\n", avgLatency)
}

func makeRequest() error {
	// client is now global and reused
	if mode == "write" {
		// Create Post
		authorID := fmt.Sprintf("bench_user_%d", rand.Intn(1000))
		payload := map[string]string{
			"author_id": authorID,
			"content":   "Load test content " + time.Now().String(),
		}
		data, _ := json.Marshal(payload)
		resp, err := client.Post(baseURL+"/posts", "application/json", bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("bad status: %d", resp.StatusCode)
		}
	} else if mode == "read" {
		// Get Feed
		userID := fmt.Sprintf("user_b_%d", rand.Intn(10)) // Simulate reading from a few users
		resp, err := client.Get(fmt.Sprintf("%s/feeds/%s", baseURL, userID))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("bad status: %d", resp.StatusCode)
		}
	}

	return nil
}
