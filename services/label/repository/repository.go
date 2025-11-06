package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Label struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	LabelKey    string    `json:"label_key"`
	LabelValue  string    `json:"label_value"`
	IsVerified  bool      `json:"is_verified"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]Label, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("label repository not initialised")
	}

	const query = `
		SELECT id,
		       user_id,
		       label_key,
		       label_value,
		       is_verified,
		       verified_at,
		       updated_at
		  FROM user_labels
		 WHERE user_id = $1
		 ORDER BY label_key ASC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user_labels: %w", err)
	}
	defer rows.Close()

	var labels []Label
	for rows.Next() {
		var lbl Label
		if err := rows.Scan(
			&lbl.ID,
			&lbl.UserID,
			&lbl.LabelKey,
			&lbl.LabelValue,
			&lbl.IsVerified,
			&lbl.VerifiedAt,
			&lbl.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("scan label: %w", err)
		}
		labels = append(labels, lbl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate labels: %w", err)
	}

	return labels, nil
}

func (r *Repository) Upsert(ctx context.Context, lbl Label) (Label, error) {
	if r == nil || r.pool == nil {
		return Label{}, fmt.Errorf("label repository not initialised")
	}
	now := time.Now().UTC()

	const query = `
		INSERT INTO user_labels (
			id,
			user_id,
			label_key,
			label_value,
			is_verified,
			verified_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, label_key) DO UPDATE SET
			label_value = EXCLUDED.label_value,
			is_verified = EXCLUDED.is_verified,
			verified_at = EXCLUDED.verified_at,
			updated_at = EXCLUDED.updated_at
		RETURNING id,
		          user_id,
		          label_key,
		          label_value,
		          is_verified,
		          verified_at,
		          updated_at
	`

	var saved Label
	err := r.pool.QueryRow(
		ctx,
		query,
		lbl.ID,
		lbl.UserID,
		lbl.LabelKey,
		lbl.LabelValue,
		lbl.IsVerified,
		lbl.VerifiedAt,
		now,
	).Scan(
		&saved.ID,
		&saved.UserID,
		&saved.LabelKey,
		&saved.LabelValue,
		&saved.IsVerified,
		&saved.VerifiedAt,
		&saved.LastUpdated,
	)
	if err != nil {
		return Label{}, fmt.Errorf("upsert label: %w", err)
	}
	return saved, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("label repository not initialised")
	}
	return r.pool.Ping(ctx)
}
