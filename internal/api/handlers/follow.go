package handlers

import (
	"encoding/json"
	"net/http"
)

type FollowRequest struct {
	FollowerID string `json:"follower_id"`
	FolloweeID string `json:"followee_id"`
}

func (h *Handlers) Follow(w http.ResponseWriter, r *http.Request) {
	var req FollowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.FolloweeID == "" || req.FollowerID == "" {
		http.Error(w, "followee_id and follower_id required", http.StatusBadRequest)
		return
	}

	if err := h.followersRepo.Follow(r.Context(), req.FollowerID, req.FolloweeID); err != nil {
		http.Error(w, "failed to follow", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"followed successfully"}`))
}
