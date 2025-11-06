package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type label struct {
	UserID      string    `json:"user_id"`
	LabelKey    string    `json:"label_key"`
	LabelValue  string    `json:"label_value"`
	IsVerified  bool      `json:"is_verified"`
	VerifiedAt  time.Time `json:"verified_at,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

func main() {
	port := getEnv("PORT", "7000")
	router := mux.NewRouter()

	router.HandleFunc("/healthz", handleHealth).Methods(http.MethodGet)
	router.HandleFunc("/v1/labels/{userId}", handleGetLabels).Methods(http.MethodGet)
	router.HandleFunc("/v1/labels", handleUpsertLabel).Methods(http.MethodPost)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Label service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "label"})
}

func handleGetLabels(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	log.Printf("fetching labels for %s", userID)

	mock := []label{
		{
			UserID:      userID,
			LabelKey:    "/affiliation",
			LabelValue:  "서울대학교",
			IsVerified:  true,
			VerifiedAt:  time.Now().Add(-24 * time.Hour),
			LastUpdated: time.Now().Add(-12 * time.Hour),
		},
		{
			UserID:      userID,
			LabelKey:    "/interest",
			LabelValue:  "운동",
			IsVerified:  false,
			LastUpdated: time.Now().Add(-2 * time.Hour),
		},
	}

	writeJSON(w, http.StatusOK, mock)
}

func handleUpsertLabel(w http.ResponseWriter, r *http.Request) {
	var payload label
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	payload.LastUpdated = time.Now().UTC()
	log.Printf("upserting label %s=%s for user %s", payload.LabelKey, payload.LabelValue, payload.UserID)

	// TODO: DB upsert 및 OpenSearch 연동
	writeJSON(w, http.StatusOK, payload)
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json error: %v", err)
	}
}
