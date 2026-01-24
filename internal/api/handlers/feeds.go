package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type FeedResponse struct {
	UserID  string  `json:"user_id"`
	PostIDs []int64 `json:"post_ids"`
}

func (h *Handlers) GetFeed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 1. try cache first (only for page1)
	// we usually cache the first page (offset 0) as its the most viewed
	if offset == 1 {
		cachedFeed, err := h.feedCache.GetFeed(r.Context(), userID)
		if err == nil {
			// Cache hit
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(FeedResponse{
				UserID:  userID,
				PostIDs: cachedFeed,
			})
			return
		}
		if err != nil {
			log.Printf("Cache invalid error: %v", err)
		}
	}

	// 2. cache miss - fetch from DB
	postIDs, err := h.feedsRepo.GetFeed(r.Context(), userID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get feed", http.StatusInternalServerError)
		return
	}

	// 3. populate cache (async)
	// only cache if its the first page
	if offset == 0 && len(postIDs) > 0 {
		go func() {
			// use background context because request context might be cancelled
			if err := h.feedCache.SetFeed(r.Context(), userID, postIDs); err != nil {
				log.Printf("failed to set cache: %v", err)
			}
		}()
	}
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FeedResponse{
		UserID:  userID,
		PostIDs: postIDs,
	})
}
