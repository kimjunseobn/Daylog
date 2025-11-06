package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type community struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	IsProOnly   bool      `json:"is_pro_only"`
	Members     int       `json:"members"`
	CreatedAt   time.Time `json:"created_at"`
}

func main() {
	port := getEnv("PORT", "7000")
	router := mux.NewRouter()

	router.HandleFunc("/healthz", handleHealth).Methods(http.MethodGet)
	router.HandleFunc("/v1/communities", handleListCommunities).Methods(http.MethodGet)
	router.HandleFunc("/v1/communities", handleCreateCommunity).Methods(http.MethodPost)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Community service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "community"})
}

func handleListCommunities(w http.ResponseWriter, _ *http.Request) {
	mock := []community{
		{
			ID:          "community-free-01",
			Title:       "아침형 인간 챌린지",
			Description: "매일 오전 6시에 일어나기 도전",
			IsProOnly:   false,
			Members:     123,
			CreatedAt:   time.Now().Add(-72 * time.Hour),
		},
		{
			ID:          "community-pro-01",
			Title:       "프로덕트 매니저 집중 스프린트",
			Description: "Pro 전용, 분기별 OKR 공유 및 피드백",
			IsProOnly:   true,
			Members:     45,
			CreatedAt:   time.Now().Add(-240 * time.Hour),
		},
	}

	writeJSON(w, http.StatusOK, mock)
}

func handleCreateCommunity(w http.ResponseWriter, r *http.Request) {
	var payload community
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	payload.CreatedAt = time.Now().UTC()
	log.Printf("creating community %s", payload.Title)

	// TODO: DB insert, entitlement 검증 추가
	writeJSON(w, http.StatusCreated, payload)
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
		log.Printf("json write error: %v", err)
	}
}
