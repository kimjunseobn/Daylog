package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Entitlement represents a row in user_entitlements.
type Entitlement struct {
	UserID             string
	Tier               string
	RenewalDate        *time.Time
	Status             string
	StripeSubscription string
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) UpsertEntitlement(ctx context.Context, ent Entitlement) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("entitlement repository not initialised")
	}

	const query = `
		INSERT INTO user_entitlements (
			user_id,
			tier,
			renewal_date,
			status,
			stripe_subscription_id
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			tier = EXCLUDED.tier,
			renewal_date = EXCLUDED.renewal_date,
			status = EXCLUDED.status,
			stripe_subscription_id = EXCLUDED.stripe_subscription_id
	`

	_, err := r.pool.Exec(ctx, query,
		ent.UserID,
		ent.Tier,
		ent.RenewalDate,
		ent.Status,
		ent.StripeSubscription,
	)
	if err != nil {
		return fmt.Errorf("upsert user_entitlements: %w", err)
	}
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, userID, status string) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("entitlement repository not initialised")
	}

	const query = `
		UPDATE user_entitlements
		   SET status = $2
		 WHERE user_id = $1
	`

	ct, err := r.pool.Exec(ctx, query, userID, status)
	if err != nil {
		return fmt.Errorf("update user_entitlements status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("entitlement not found for user %s", userID)
	}
	return nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("entitlement repository not initialised")
	}
	return r.pool.Ping(ctx)
}

func (r *Repository) GetByUser(ctx context.Context, userID string) (*Entitlement, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("entitlement repository not initialised")
	}

	const query = `
		SELECT user_id,
		       tier,
		       renewal_date,
		       status,
		       stripe_subscription_id
		  FROM user_entitlements
		 WHERE user_id = $1
	`

	var ent Entitlement
	var renewal *time.Time
	if err := r.pool.QueryRow(ctx, query, userID).Scan(
		&ent.UserID,
		&ent.Tier,
		&renewal,
		&ent.Status,
		&ent.StripeSubscription,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ent.RenewalDate = renewal
	return &ent, nil
}
