package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const baseURL = "http://localhost:8080"

func main() {
	// 1. Generate unique User IDs
	userA := fmt.Sprintf("user_a_%d", time.Now().Unix()) // The Author
	userB := fmt.Sprintf("user_b_%d", time.Now().Unix()) // The Follower

	fmt.Printf("🧪 Starting E2E Test\n")
	fmt.Printf("   Author: %s\n   Follower: %s\n\n", userA, userB)

	// 2. UserB follows UserA
	// POST /follow
	fmt.Println("👉 Step 1: UserB follows UserA...")
	sendRequest("POST", "/follow", map[string]string{
		"follower_id": userB,
		"followee_id": userA,
	})

	// 3. UserA creates a Post
	// POST /posts
	fmt.Println("👉 Step 2: UserA creates a Post...")
	postPayload := map[string]string{
		"author_id": userA,
		"content":   "Hello World from E2E Test!",
	}
	resp := sendRequest("POST", "/posts", postPayload)

	var postResp struct {
		PostID  int64  `json:"post_id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(resp), &postResp); err != nil {
		log.Fatalf("Failed to parse create post response: %v", err)
	}
	postID := postResp.PostID
	fmt.Printf("   Created Post ID: %d\n", postID)

	// 4. Wait for Asynchronous Processing
	// API -> Kafka -> Processor -> DB
	fmt.Println("⏳ Step 3: Waiting 3 seconds for async processing...")
	time.Sleep(3 * time.Second)

	// 5. Check UserB's Feed
	// GET /feeds/{id}
	fmt.Println("👉 Step 4: Checking UserB's Feed...")
	feedBody := sendRequest("GET", fmt.Sprintf("/feeds/%s", userB), nil)

	var feedResp struct {
		UserID  string  `json:"user_id"`
		PostIDs []int64 `json:"post_ids"`
	}
	if err := json.Unmarshal([]byte(feedBody), &feedResp); err != nil {
		log.Fatalf("Failed to parse feed response: %v", err)
	}

	// 6. Assertions
	found := false
	expectedID := postID
	for _, id := range feedResp.PostIDs {
		if id == expectedID {
			found = true
			break
		}
	}

	if found {
		fmt.Println("\n✅ SUCCESS: Post found in UserB's feed!")
	} else {
		fmt.Printf("\n❌ FAILURE: Post %v NOT found in feed %v\n", expectedID, feedResp.PostIDs)
		fmt.Println("   Possible issues: Kafka consumer lag, processor crash, or DB insert failed.")
		log.Fatal("Test Failed")
	}
}

func sendRequest(method, endpoint string, body interface{}) string {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(data)
	}

	req, _ := http.NewRequest(method, baseURL+endpoint, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Fatalf("API Error (%d): %s", resp.StatusCode, string(respBytes))
	}

	return string(respBytes)
}
