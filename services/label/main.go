package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"daylog/services/common/config"
	"daylog/services/common/db"
	"daylog/services/common/logging"
	"daylog/services/label/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type server struct {
	cfg    config.Config
	logger *zap.SugaredLogger
	repo   *repository.Repository
	router *mux.Router
}

type labelRequest struct {
	UserID     string  `json:"user_id"`
	LabelKey   string  `json:"label_key"`
	LabelValue string  `json:"label_value"`
	IsVerified *bool   `json:"is_verified,omitempty"`
	VerifiedAt *string `json:"verified_at,omitempty"`
}

func main() {
	cfg := config.MustLoad("label")

	logger, err := logging.Init(cfg.Service.Name, cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	if !cfg.HasPostgres() {
		logger.Fatal("POSTGRES_URI must be set for label service")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.Postgres.URI)
	if err != nil {
		logger.Fatalw("failed to connect postgres", "error", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	srv := newServer(cfg, logger, repo)

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Errorw("failed to shutdown http server", "error", err)
		}
	}()

	logger.Infow("label service listening", "addr", cfg.Addr())
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalw("http server error", "error", err)
	}
}

func newServer(cfg config.Config, logger *zap.SugaredLogger, repo *repository.Repository) *server {
	s := &server{
		cfg:    cfg,
		logger: logger,
		repo:   repo,
		router: mux.NewRouter(),
	}

	s.router.Use(s.loggingMiddleware)
	s.router.HandleFunc("/healthz", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/readyz", s.handleReady).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/labels/{userId}", s.handleListLabels).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/labels", s.handleUpsertLabel).Methods(http.MethodPost)

	return s
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": s.cfg.Service.Name,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	if err := s.repo.Ping(ctx); err != nil {
		status = err.Error()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": s.cfg.Service.Name,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"checks": map[string]string{
			"postgres": status,
		},
	})
}

func (s *server) handleListLabels(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	labels, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		s.logger.Errorw("failed to list labels", "error", err, "user_id", userID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch labels"})
		return
	}

	writeJSON(w, http.StatusOK, labels)
}

func (s *server) handleUpsertLabel(w http.ResponseWriter, r *http.Request) {
	var payload labelRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	if err := validateLabel(payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var (
		verifiedAt *time.Time
	)
	if payload.VerifiedAt != nil && *payload.VerifiedAt != "" {
		t, err := time.Parse(time.RFC3339, *payload.VerifiedAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid verified_at format"})
			return
		}
		verifiedAt = &t
	}

	isVerified := false
	if payload.IsVerified != nil {
		isVerified = *payload.IsVerified
	}

	labelID := uuid.NewString()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	saved, err := s.repo.Upsert(ctx, repository.Label{
		ID:         labelID,
		UserID:     payload.UserID,
		LabelKey:   normaliseKey(payload.LabelKey),
		LabelValue: payload.LabelValue,
		IsVerified: isVerified,
		VerifiedAt: verifiedAt,
	})
	if err != nil {
		s.logger.Errorw("failed to upsert label", "error", err, "user_id", payload.UserID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store label"})
		return
	}

	writeJSON(w, http.StatusOK, saved)
}

func (s *server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Infow("request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func validateLabel(payload labelRequest) error {
	if payload.UserID == "" {
		return errors.New("user_id is required")
	}
	if payload.LabelKey == "" {
		return errors.New("label_key is required")
	}
	if payload.LabelValue == "" {
		return errors.New("label_value is required")
	}
	return nil
}

func normaliseKey(key string) string {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "/") {
		key = "/" + key
	}
	return key
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
