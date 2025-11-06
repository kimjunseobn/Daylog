package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type feedItem struct {
	PostID     string    `json:"post_id"`
	UserID     string    `json:"user_id"`
	TimelineID string    `json:"timeline_id"`
	Category   string    `json:"category"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

func main() {
	port := getEnv("PORT", "7000")
	router := mux.NewRouter()

	router.HandleFunc("/healthz", handleHealth).Methods(http.MethodGet)
	router.HandleFunc("/v1/feed/{userId}", handleFeed).Methods(http.MethodGet)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Social feed service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "social-feed",
	})
}

func handleFeed(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	log.Printf("building social feed for %s", userID)

	mock := []feedItem{
		{
			PostID:     "post-123",
			UserID:     userID,
			TimelineID: "timeline-abc",
			Category:   "work",
			Message:    "오늘은 생산적인 하루였어요!",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			PostID:     "post-456",
			UserID:     "peer-789",
			TimelineID: "timeline-def",
			Category:   "exercise",
			Message:    "저녁에는 운동하며 마무리했어요.",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
		},
	}

	writeJSON(w, http.StatusOK, mock)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}
