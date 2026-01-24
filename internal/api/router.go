package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/its-me-ojas/event-driven-feed/internal/api/handlers"
	"github.com/its-me-ojas/event-driven-feed/internal/api/middleware"
)

func NewRouter(h *handlers.Handlers) *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	r.HandleFunc("/posts", h.CreatePost).Methods("POST")
	r.HandleFunc("/feeds/{user_id}", h.GetFeed).Methods("GET")
	r.HandleFunc("/follow", h.Follow).Methods("POST")

	return r
}
