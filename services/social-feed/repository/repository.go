package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedItem struct {
	PostID     string                 `json:"post_id"`
	UserID     string                 `json:"user_id"`
	TimelineID string                 `json:"timeline_id"`
	Category   string                 `json:"category"`
	Message    string                 `json:"message"`
	CreatedAt  time.Time              `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListByUser(ctx context.Context, userID string, limit int) ([]FeedItem, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("feed repository not initialised")
	}

	const query = `
		SELECT post_id,
		       user_id,
		       timeline_id,
		       category,
		       message,
		       created_at,
		       metadata
		  FROM social_posts
		 WHERE user_id = $1
		    OR user_id IN (
				SELECT follows_user_id
				  FROM social_relationships
				 WHERE user_id = $1
			)
		 ORDER BY created_at DESC
		 LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query social_posts: %w", err)
	}
	defer rows.Close()

	items := make([]FeedItem, 0, limit)
	for rows.Next() {
		var (
			item        FeedItem
			metadataRaw []byte
		)
		if err := rows.Scan(
			&item.PostID,
			&item.UserID,
			&item.TimelineID,
			&item.Category,
			&item.Message,
			&item.CreatedAt,
			&metadataRaw,
		); err != nil {
			return nil, fmt.Errorf("scan feed item: %w", err)
		}
		if len(metadataRaw) > 0 {
			if err := json.Unmarshal(metadataRaw, &item.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal feed metadata: %w", err)
			}
		} else {
			item.Metadata = map[string]interface{}{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feed rows: %w", err)
	}
	return items, nil
}

func (r *Repository) Create(ctx context.Context, item FeedItem) (FeedItem, error) {
	if r == nil || r.pool == nil {
		return FeedItem{}, fmt.Errorf("feed repository not initialised")
	}

	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return FeedItem{}, fmt.Errorf("marshal metadata: %w", err)
	}

	const query = `
		INSERT INTO social_posts (
			post_id,
			user_id,
			timeline_id,
			category,
			message,
			metadata,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING post_id,
		          user_id,
		          timeline_id,
		          category,
		          message,
		          metadata,
		          created_at
	`

	var saved FeedItem
	err = r.pool.QueryRow(
		ctx,
		query,
		item.PostID,
		item.UserID,
		item.TimelineID,
		item.Category,
		item.Message,
		metadata,
		time.Now().UTC(),
	).Scan(
		&saved.PostID,
		&saved.UserID,
		&saved.TimelineID,
		&saved.Category,
		&saved.Message,
		&metadata,
		&saved.CreatedAt,
	)
	if err != nil {
		return FeedItem{}, fmt.Errorf("insert social_post: %w", err)
	}

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &saved.Metadata); err != nil {
			return FeedItem{}, fmt.Errorf("unmarshal saved metadata: %w", err)
		}
	} else {
		saved.Metadata = map[string]interface{}{}
	}

	return saved, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("feed repository not initialised")
	}
	return r.pool.Ping(ctx)
}
