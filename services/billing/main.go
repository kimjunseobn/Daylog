package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"daylog/services/billing/repository"
	"daylog/services/common/config"
	"daylog/services/common/db"
	"daylog/services/common/logging"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"go.uber.org/zap"
)

type server struct {
	cfg          config.Config
	logger       *zap.SugaredLogger
	repo         *repository.Repository
	stripeSecret string
	router       *mux.Router
}

type webhookAck struct {
	Status    string `json:"status"`
	Processed bool   `json:"processed"`
}

func main() {
	cfg := config.MustLoad("billing")

	logger, err := logging.Init(cfg.Service.Name, cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer logging.Sync()

	if !cfg.HasPostgres() {
		logger.Fatal("POSTGRES_URI must be set for billing service")
	}
	if !cfg.HasStripeWebhook() {
		logger.Fatal("STRIPE_WEBHOOK_SECRET must be set for billing service")
	}

	if cfg.Stripe.APIKey != "" {
		stripe.Key = cfg.Stripe.APIKey
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

	logger.Infow("billing service listening", "addr", cfg.Addr())
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalw("http server error", "error", err)
	}
}

func newServer(cfg config.Config, logger *zap.SugaredLogger, repo *repository.Repository) *server {
	s := &server{
		cfg:          cfg,
		logger:       logger,
		repo:         repo,
		stripeSecret: cfg.Stripe.WebhookSecret,
		router:       mux.NewRouter(),
	}

	s.router.Use(s.loggingMiddleware)
	s.router.HandleFunc("/healthz", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/readyz", s.handleReady).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/entitlements/{userId}", s.handleGetEntitlement).Methods(http.MethodGet)
	s.router.HandleFunc("/v1/webhooks/stripe", s.handleStripeWebhook).Methods(http.MethodPost)

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
			"stripe":   boolToStatus(s.stripeSecret != ""),
		},
	})
}

func (s *server) handleGetEntitlement(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userId"]
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ent, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		s.logger.Errorw("failed to fetch entitlement", "user_id", userID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch entitlement"})
		return
	}
	if ent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, ent)
}

func (s *server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if s.stripeSecret == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stripe webhook secret missing"})
		return
	}

	bodyReader := http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	defer bodyReader.Close()

	payload, err := io.ReadAll(bodyReader)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request"})
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, s.stripeSecret)
	if err != nil {
		s.logger.Warnw("stripe signature verification failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "signature verification failed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := s.processEvent(ctx, event); err != nil {
		s.logger.Errorw("failed to process stripe event", "event_id", event.ID, "type", event.Type, "error", err)
		writeJSON(w, http.StatusOK, webhookAck{Status: "error", Processed: false})
		return
	}

	writeJSON(w, http.StatusOK, webhookAck{Status: "ok", Processed: true})
}

func (s *server) processEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			return err
		}
		return s.handleSubscription(ctx, &subscription)

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			return err
		}
		userID := subscription.Metadata["user_id"]
		if userID == "" {
			s.logger.Warnw("subscription deleted without user_id metadata", "subscription_id", subscription.ID)
			return nil
		}
		status := string(subscription.Status)
		if status == "" {
			status = "canceled"
		}
		return s.repo.UpdateStatus(ctx, userID, status)

	case "invoice.paid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return err
		}
		return s.handleInvoicePaid(ctx, &invoice)

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return err
		}
		return s.handlePaymentFailure(ctx, &invoice)

	default:
		s.logger.Infow("received unhandled stripe event", "event_type", event.Type)
		return nil
	}
}

func (s *server) handleSubscription(ctx context.Context, subscription *stripe.Subscription) error {
	userID := subscription.Metadata["user_id"]
	if userID == "" {
		s.logger.Warnw("subscription event missing user_id metadata", "subscription_id", subscription.ID)
		return nil
	}

	tier := subscription.Metadata["tier"]
	if tier == "" {
		tier = "pro"
	}

	var renewal *time.Time
	if subscription.CurrentPeriodEnd > 0 {
		t := time.Unix(subscription.CurrentPeriodEnd, 0).UTC()
		renewal = &t
	}

	ent := repository.Entitlement{
		UserID:             userID,
		Tier:               tier,
		RenewalDate:        renewal,
		Status:             string(subscription.Status),
		StripeSubscription: subscription.ID,
	}

	return s.repo.UpsertEntitlement(ctx, ent)
}

func (s *server) handleInvoicePaid(ctx context.Context, invoice *stripe.Invoice) error {
	userID := invoice.Metadata["user_id"]
	if userID == "" {
		// fall back to subscription metadata if expanded
		if invoice.Subscription != nil {
			userID = invoice.Subscription.Metadata["user_id"]
		}
	}
	if userID == "" {
		s.logger.Warnw("invoice paid without user metadata", "invoice_id", invoice.ID)
		return nil
	}

	tier := invoice.Metadata["tier"]
	if tier == "" && invoice.Subscription != nil {
		tier = invoice.Subscription.Metadata["tier"]
	}
	if tier == "" {
		tier = "pro"
	}

	var renewal *time.Time
	if invoice.Subscription != nil && invoice.Subscription.CurrentPeriodEnd > 0 {
		t := time.Unix(invoice.Subscription.CurrentPeriodEnd, 0).UTC()
		renewal = &t
	} else if len(invoice.Lines.Data) > 0 && invoice.Lines.Data[0].Period.End > 0 {
		t := time.Unix(invoice.Lines.Data[0].Period.End, 0).UTC()
		renewal = &t
	}

	ent := repository.Entitlement{
		UserID:             userID,
		Tier:               tier,
		RenewalDate:        renewal,
		Status:             "active",
		StripeSubscription: invoice.SubscriptionID,
	}

	return s.repo.UpsertEntitlement(ctx, ent)
}

func (s *server) handlePaymentFailure(ctx context.Context, invoice *stripe.Invoice) error {
	userID := invoice.Metadata["user_id"]
	if userID == "" && invoice.Subscription != nil {
		userID = invoice.Subscription.Metadata["user_id"]
	}
	if userID == "" {
		s.logger.Warnw("payment failure without user metadata", "invoice_id", invoice.ID)
		return nil
	}

	tier := invoice.Metadata["tier"]
	if tier == "" {
		tier = "pro"
	}
	ent := repository.Entitlement{
		UserID:             userID,
		Tier:               tier,
		Status:             "past_due",
		StripeSubscription: invoice.SubscriptionID,
	}
	return s.repo.UpsertEntitlement(ctx, ent)
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

func boolToStatus(v bool) string {
	if v {
		return "ok"
	}
	return "disabled"
}
