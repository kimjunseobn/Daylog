package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type webhookAck struct {
	Status    string `json:"status"`
	Processed bool   `json:"processed"`
}

func main() {
	port := getEnv("PORT", "7000")
	router := mux.NewRouter()

	router.HandleFunc("/healthz", handleHealth).Methods(http.MethodGet)
	router.HandleFunc("/v1/webhooks/stripe", handleStripeWebhook).Methods(http.MethodPost)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Billing service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "billing",
	})
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	defer r.Body.Close()

	// TODO: Stripe 시그니처 검증 및 이벤트 처리
	log.Printf("received stripe webhook: %s", string(payload))
	writeJSON(w, http.StatusOK, webhookAck{Status: "ok", Processed: true})
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
		log.Printf("json encode error: %v", err)
	}
}
