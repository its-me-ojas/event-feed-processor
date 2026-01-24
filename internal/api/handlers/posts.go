package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/its-me-ojas/event-driven-feed/internal/events"
)

type CreatePostRequest struct {
	AuthorID string `json:"author_id"`
	Content  string `json:"content"`
}

type CreatePostResponse struct {
	PostID  int64  `json:"post_id"`
	Message string `json:"message"`
}

func (h *Handlers) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AuthorID == "" || req.Content == "" {
		http.Error(w, "author_id and content required", http.StatusBadRequest)
		return
	}

	eventID := h.idGen.Generate()
	postID := h.idGen.Generate()

	event := events.NewPostCreatedEvent(eventID, postID, req.AuthorID, req.Content)
	data, err := event.Marshal()
	if err != nil {
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	if err := h.producer.Publish(r.Context(), req.AuthorID, data); err != nil {
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(CreatePostResponse{
		PostID:  postID,
		Message: "Post created successfully",
	})
}
