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
	"daylog/services/common/messaging"
	"daylog/services/ingestion/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type server struct {
	cfg      config.Config
	logger   *zap.SugaredLogger
	producer *messaging.Producer
	repo     *repository.EventRepository
	router   *mux.Router
}

type activityEvent struct {
	EventID   string                 `json:"event_id,omitempty"`
	UserID    string                 `json:"user_id"`
	Source    string                 `json:"source"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   time.Time              `json:"ended_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type healthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Time    time.Time         `json:"time"`
	Checks  map[string]string `json:"checks,omitempty"`
}

func main() {
	cfg := config.MustLoad("ingestion")

	logger, err := logging.Init(cfg.Service.Name, cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var (
		pool *repository.EventRepository
	)

	if cfg.HasPostgres() {
		pgPool, err := db.NewPool(ctx, cfg.Postgres.URI)
		if err != nil {
			logger.Fatalw("failed to create postgres pool", "error", err)
		}
		defer pgPool.Close()
		pool = repository.NewEventRepository(pgPool)
	} else {
		logger.Warn("POSTGRES_URI not set, raw events will not be persisted")
	}

	var producer *messaging.Producer
	if cfg.HasKafka() {
		producer, err = messaging.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.ActivityTopic, logger)
		if err != nil {
			logger.Fatalw("failed to initialise kafka producer", "error", err)
		}
		defer producer.Close()
	} else {
		logger.Warn("KAFKA_BROKERS not set, events will not be published to Kafka")
	}

	srv := newServer(cfg, logger, producer, pool)

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

	logger.Infow("ingestion service listening", "addr", cfg.Addr())
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalw("http server error", "error", err)
	}
}

func newServer(
	cfg config.Config,
	logger *zap.SugaredLogger,
	producer *messaging.Producer,
	repo *repository.EventRepository,
) *server {
	s := &server{
		cfg:      cfg,
		logger:   logger,
		producer: producer,
		repo:     repo,
		router:   mux.NewRouter(),
	}

	s.router.Use(s.loggingMiddleware)
	s.router.HandleFunc("/healthz", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/readyz", s.handleReady).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/events", s.handleEventIngestion).Methods(http.MethodPost)

	return s
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "ok",
		Service: s.cfg.Service.Name,
		Time:    time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{}

	if s.repo != nil {
		if err := s.repo.Ping(ctx); err != nil {
			checks["postgres"] = err.Error()
		} else {
			checks["postgres"] = "ok"
		}
	} else {
		checks["postgres"] = "disabled"
	}

	if s.producer != nil {
		checks["kafka"] = "ok"
	} else {
		checks["kafka"] = "disabled"
	}

	resp := healthResponse{
		Status:  "ok",
		Service: s.cfg.Service.Name,
		Time:    time.Now().UTC(),
		Checks:  checks,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleEventIngestion(w http.ResponseWriter, r *http.Request) {
	var payload activityEvent
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	if payload.Metadata == nil {
		payload.Metadata = map[string]interface{}{}
	}

	eventID := uuid.NewString()
	payload.EventID = eventID

	if s.producer != nil {
		bytes, err := json.Marshal(payload)
		if err != nil {
			s.logger.Errorw("failed to marshal payload for kafka", "error", err)
		} else {
			if err := s.producer.Publish(r.Context(), []byte(payload.UserID), bytes); err != nil {
				s.logger.Errorw("failed to publish kafka message", "error", err)
			}
		}
	}

	if s.repo != nil {
		err := s.repo.Save(r.Context(), repository.Event{
			EventID:        eventID,
			UserID:         payload.UserID,
			Source:         payload.Source,
			TimestampStart: payload.StartedAt,
			TimestampEnd:   payload.EndedAt,
			Metadata:       payload.Metadata,
		})
		if err != nil {
			s.logger.Errorw("failed to persist activity event", "error", err)
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued", "event_id": eventID})
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
