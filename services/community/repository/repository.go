package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Community struct {
	ID          string    `json:"id"`
	AccessLevel string    `json:"access_level"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	IsProOnly   bool      `json:"is_pro_only"`
	CreatedAt   time.Time `json:"created_at"`
}

type Membership struct {
	CommunityID string    `json:"community_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListCommunities(ctx context.Context, includePro bool) ([]Community, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("community repository not initialised")
	}

	query := `
		SELECT community_id,
		       access_level,
		       title,
		       description,
		       is_pro_only,
		       created_at
		  FROM communities
	`
	if !includePro {
		query += " WHERE is_pro_only = false"
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query communities: %w", err)
	}
	defer rows.Close()

	var items []Community
	for rows.Next() {
		var c Community
		if err := rows.Scan(
			&c.ID,
			&c.AccessLevel,
			&c.Title,
			&c.Description,
			&c.IsProOnly,
			&c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan community: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate communities: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateCommunity(ctx context.Context, c Community) (Community, error) {
	if r == nil || r.pool == nil {
		return Community{}, fmt.Errorf("community repository not initialised")
	}

	const query = `
		INSERT INTO communities (
			community_id,
			access_level,
			title,
			description,
			is_pro_only,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING community_id,
		          access_level,
		          title,
		          description,
		          is_pro_only,
		          created_at
	`

	var saved Community
	if err := r.pool.QueryRow(
		ctx,
		query,
		c.ID,
		c.AccessLevel,
		c.Title,
		c.Description,
		c.IsProOnly,
		time.Now().UTC(),
	).Scan(
		&saved.ID,
		&saved.AccessLevel,
		&saved.Title,
		&saved.Description,
		&saved.IsProOnly,
		&saved.CreatedAt,
	); err != nil {
		return Community{}, fmt.Errorf("insert community: %w", err)
	}

	return saved, nil
}

func (r *Repository) JoinCommunity(ctx context.Context, userID, communityID, role string) (Membership, error) {
	if r == nil || r.pool == nil {
		return Membership{}, fmt.Errorf("community repository not initialised")
	}

	const query = `
		INSERT INTO community_memberships (
			community_id,
			user_id,
			role,
			joined_at
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (community_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			joined_at = EXCLUDED.joined_at
		RETURNING community_id,
		          user_id,
		          role,
		          joined_at
	`

	var membership Membership
	if err := r.pool.QueryRow(
		ctx,
		query,
		communityID,
		userID,
		role,
		time.Now().UTC(),
	).Scan(
		&membership.CommunityID,
		&membership.UserID,
		&membership.Role,
		&membership.JoinedAt,
	); err != nil {
		return Membership{}, fmt.Errorf("upsert membership: %w", err)
	}

	return membership, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("community repository not initialised")
	}
	return r.pool.Ping(ctx)
}
