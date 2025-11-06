package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"daylog/services/common/config"
	"daylog/services/common/db"
	"daylog/services/common/logging"
	"daylog/services/community/repository"

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

type createCommunityRequest struct {
	AccessLevel string `json:"access_level"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IsProOnly   bool   `json:"is_pro_only"`
}

type joinRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func main() {
	cfg := config.MustLoad("community")

	logger, err := logging.Init(cfg.Service.Name, cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	if !cfg.HasPostgres() {
		logger.Fatal("POSTGRES_URI must be set for community service")
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

	logger.Infow("community service listening", "addr", cfg.Addr())
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
	s.router.HandleFunc("/v1/communities", s.handleListCommunities).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/communities", s.handleCreateCommunity).Methods(http.MethodPost)
	s.router.HandleFunc("/v1/communities/{communityId}/join", s.handleJoinCommunity).Methods(http.MethodPost)

	return s
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
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

func (s *server) handleListCommunities(w http.ResponseWriter, r *http.Request) {
	includePro := r.URL.Query().Get("include_pro") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	communities, err := s.repo.ListCommunities(ctx, includePro)
	if err != nil {
		s.logger.Errorw("failed to list communities", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch communities"})
		return
	}

	writeJSON(w, http.StatusOK, communities)
}

func (s *server) handleCreateCommunity(w http.ResponseWriter, r *http.Request) {
	var payload createCommunityRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	if payload.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	if payload.AccessLevel == "" {
		payload.AccessLevel = "public"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	community, err := s.repo.CreateCommunity(ctx, repository.Community{
		ID:          uuid.NewString(),
		AccessLevel: payload.AccessLevel,
		Title:       payload.Title,
		Description: payload.Description,
		IsProOnly:   payload.IsProOnly,
	})
	if err != nil {
		s.logger.Errorw("failed to create community", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create community"})
		return
	}

	writeJSON(w, http.StatusCreated, community)
}

func (s *server) handleJoinCommunity(w http.ResponseWriter, r *http.Request) {
	var payload joinRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	if payload.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		return
	}
	role := payload.Role
	if role == "" {
		role = "member"
	}

	communityID := mux.Vars(r)["communityId"]
	if communityID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "communityId is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	membership, err := s.repo.JoinCommunity(ctx, payload.UserID, communityID, role)
	if err != nil {
		s.logger.Errorw("failed to join community", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to join community"})
		return
	}

	writeJSON(w, http.StatusOK, membership)
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
