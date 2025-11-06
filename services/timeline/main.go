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
	"daylog/services/timeline/repository"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type server struct {
	cfg      config.Config
	logger   *zap.SugaredLogger
	repo     *repository.Repository
	consumer *messaging.Consumer
	router   *mux.Router
}

type activityEvent struct {
	EventID   string                 `json:"event_id"`
	UserID    string                 `json:"user_id"`
	Source    string                 `json:"source"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   time.Time              `json:"ended_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func main() {
	cfg := config.MustLoad("timeline")

	logger, err := logging.Init(cfg.Service.Name, cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !cfg.HasPostgres() {
		logger.Fatal("POSTGRES_URI must be set for timeline service")
	}

	pool, err := db.NewPool(ctx, cfg.Postgres.URI)
	if err != nil {
		logger.Fatalw("failed to connect postgres", "error", err)
	}
	defer pool.Close()

	repo := repository.New(pool)

	var consumer *messaging.Consumer
	if cfg.HasKafka() {
		consumer, err = messaging.NewConsumer(messaging.ConsumerConfig{
			Brokers: cfg.Kafka.Brokers,
			Topic:   cfg.Kafka.ActivityTopic,
			GroupID: cfg.Kafka.GroupID,
		}, logger)
		if err != nil {
			logger.Errorw("failed to initialise kafka consumer", "error", err)
		} else {
			go startConsumerLoop(ctx, logger, consumer, repo)
		}
	} else {
		logger.Warn("timeline consumer disabled: KAFKA_BROKERS not set")
	}

	srv := newServer(cfg, logger, repo, consumer)

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
		if consumer != nil {
			_ = consumer.Close()
		}
	}()

	logger.Infow("timeline service listening", "addr", cfg.Addr())
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalw("http server error", "error", err)
	}
}

func newServer(cfg config.Config, logger *zap.SugaredLogger, repo *repository.Repository, consumer *messaging.Consumer) *server {
	s := &server{
		cfg:      cfg,
		logger:   logger,
		repo:     repo,
		consumer: consumer,
		router:   mux.NewRouter(),
	}

	s.router.Use(s.loggingMiddleware)
	s.router.HandleFunc("/healthz", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/readyz", s.handleReady).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/timeline/{userId}", s.handleGetTimeline).Methods(http.MethodGet)

	return s
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
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

	if s.consumer != nil {
		status["kafka"] = "ok"
	} else {
		status["kafka"] = "disabled"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": s.cfg.Service.Name,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"checks":  status,
	})
}

func (s *server) handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId is required"})
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	entries, err := s.repo.ListActivityEvents(ctx, userID, limit)
	if err != nil {
		s.logger.Errorw("failed to fetch timeline", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch timeline"})
		return
	}

	writeJSON(w, http.StatusOK, entries)
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

func startConsumerLoop(ctx context.Context, logger *zap.SugaredLogger, consumer *messaging.Consumer, repo *repository.Repository) {
	logger.Infow("starting timeline consumer loop")
	for {
		select {
		case <-ctx.Done():
			logger.Infow("timeline consumer context cancelled")
			return
		default:
		}

		msg, err := consumer.Fetch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Infow("timeline consumer stopped")
				return
			}
			logger.Errorw("failed to fetch kafka message", "error", err)
			time.Sleep(time.Second)
			continue
		}

		var evt activityEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			logger.Errorw("failed to decode kafka message", "error", err)
			_ = consumer.Commit(ctx, msg)
			continue
		}

		entry := repository.Entry{
			EventID:      evt.EventID,
			UserID:       evt.UserID,
			Source:       evt.Source,
			Category:     evt.Source,
			StartedAt:    evt.StartedAt,
			EndedAt:      evt.EndedAt,
			Metadata:     evt.Metadata,
			GeoContext:   map[string]any{},
			Confidence:   0.6,
			SourceEvents: []string{evt.EventID},
		}

		if err := repo.UpsertTimelineEntry(ctx, entry); err != nil {
			logger.Errorw("failed to upsert timeline entry", "error", err)
		}

		if err := consumer.Commit(ctx, msg); err != nil {
			logger.Errorw("failed to commit kafka message", "error", err)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
