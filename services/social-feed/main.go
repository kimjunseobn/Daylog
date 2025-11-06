package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"daylog/services/common/config"
	"daylog/services/common/db"
	"daylog/services/common/logging"
	"daylog/services/common/messaging"
	"daylog/services/socialfeed/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type server struct {
	cfg      config.Config
	logger   *zap.SugaredLogger
	repo     *repository.Repository
	producer *messaging.Producer
	router   *mux.Router
}

type createPostRequest struct {
	UserID     string                 `json:"user_id"`
	TimelineID string                 `json:"timeline_id"`
	Category   string                 `json:"category"`
	Message    string                 `json:"message"`
	Metadata   map[string]interface{} `json:"metadata"`
}

func main() {
	cfg := config.MustLoad("social-feed")

	logger, err := logging.Init(cfg.Service.Name, cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	if !cfg.HasPostgres() {
		logger.Fatal("POSTGRES_URI must be set for social-feed service")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.Postgres.URI)
	if err != nil {
		logger.Fatalw("failed to connect postgres", "error", err)
	}
	defer pool.Close()

	repo := repository.New(pool)

	var producer *messaging.Producer
	if cfg.HasKafka() {
		producer, err = messaging.NewProducer(cfg.Kafka.Brokers, "social.feed.events", logger)
		if err != nil {
			logger.Errorw("failed to create kafka producer", "error", err)
		}
		defer func() {
			if producer != nil {
				_ = producer.Close()
			}
		}()
	}

	srv := newServer(cfg, logger, repo, producer)

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

	logger.Infow("social feed service listening", "addr", cfg.Addr())
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalw("http server error", "error", err)
	}
}

func newServer(cfg config.Config, logger *zap.SugaredLogger, repo *repository.Repository, producer *messaging.Producer) *server {
	s := &server{
		cfg:      cfg,
		logger:   logger,
		repo:     repo,
		producer: producer,
		router:   mux.NewRouter(),
	}

	s.router.Use(s.loggingMiddleware)
	s.router.HandleFunc("/healthz", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/readyz", s.handleReady).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/feed/{userId}", s.handleGetFeed).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/feed", s.handleCreatePost).Methods(http.MethodPost)

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

	status := map[string]string{}
	if err := s.repo.Ping(ctx); err != nil {
		status["postgres"] = err.Error()
	} else {
		status["postgres"] = "ok"
	}
	if s.producer != nil {
		status["kafka_producer"] = "ok"
	} else {
		status["kafka_producer"] = "disabled"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": s.cfg.Service.Name,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"checks":  status,
	})
}

func (s *server) handleGetFeed(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId is required"})
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := s.repo.ListByUser(ctx, userID, limit)
	if err != nil {
		s.logger.Errorw("failed to list feed items", "error", err, "user_id", userID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch feed"})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var payload createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	if payload.UserID == "" || payload.TimelineID == "" || payload.Category == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id, timeline_id, category are required"})
		return
	}

	if payload.Metadata == nil {
		payload.Metadata = map[string]interface{}{}
	}

	post := repository.FeedItem{
		PostID:     uuid.NewString(),
		UserID:     payload.UserID,
		TimelineID: payload.TimelineID,
		Category:   payload.Category,
		Message:    payload.Message,
		Metadata:   payload.Metadata,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	saved, err := s.repo.Create(ctx, post)
	if err != nil {
		s.logger.Errorw("failed to create feed post", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
		return
	}

	if s.producer != nil {
		bytes, _ := json.Marshal(saved)
		if err := s.producer.Publish(ctx, []byte(saved.UserID), bytes); err != nil {
			s.logger.Warnw("failed to publish social feed event", "error", err)
		}
	}

	writeJSON(w, http.StatusCreated, saved)
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
