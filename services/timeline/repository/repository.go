package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry는 타임라인 응답에 사용되는 구조체입니다.
type Entry struct {
	EventID       string                 `json:"event_id"`
	UserID        string                 `json:"user_id"`
	Category      string                 `json:"category"`
	StartedAt     time.Time              `json:"started_at"`
	EndedAt       time.Time              `json:"ended_at"`
	Confidence    float64                `json:"confidence"`
	GeoContext    map[string]any         `json:"geo_context"`
	Source        string                 `json:"source"`
	Metadata      map[string]interface{} `json:"metadata"`
	SourceEvents  []string               `json:"source_event_ids"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListActivityEvents는 activity_events 테이블을 기반으로 타임라인을 구성합니다.
func (r *Repository) ListActivityEvents(ctx context.Context, userID string, limit int) ([]Entry, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("timeline repository not initialised")
	}

	const query = `
		SELECT event_id,
		       user_id,
		       source,
		       timestamp_start,
		       timestamp_end,
		       metadata
		  FROM activity_events
		 WHERE user_id = $1
		 ORDER BY timestamp_start DESC
		 LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query activity_events: %w", err)
	}
	defer rows.Close()

	entries := make([]Entry, 0, limit)
	for rows.Next() {
		var (
			entry        Entry
			metadataJSON []byte
		)
		if err := rows.Scan(
			&entry.EventID,
			&entry.UserID,
			&entry.Source,
			&entry.StartedAt,
			&entry.EndedAt,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan activity_events row: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &entry.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		} else {
			entry.Metadata = map[string]interface{}{}
		}

		entry.Category = deriveCategory(entry)
		entry.Confidence = deriveConfidence(entry)
		entry.GeoContext = deriveGeoContext(entry.Metadata)
		entry.SourceEvents = []string{entry.EventID}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity_events: %w", err)
	}

	return entries, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("timeline repository not initialised")
	}
	return r.pool.Ping(ctx)
}

func deriveCategory(entry Entry) string {
	if cats, ok := entry.Metadata["category"].(string); ok && cats != "" {
		return cats
	}
	return entry.Source
}

func deriveConfidence(entry Entry) float64 {
	if conf, ok := entry.Metadata["confidence"].(float64); ok {
		return conf
	}
	return 0.5
}

func deriveGeoContext(metadata map[string]interface{}) map[string]any {
	if ctx, ok := metadata["geo_context"].(map[string]any); ok {
		return ctx
	}
	return map[string]any{}
}

// UpsertTimelineEntry는 timeline_entries 테이블에 기본 데이터를 저장합니다.
func (r *Repository) UpsertTimelineEntry(ctx context.Context, entry Entry) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("timeline repository not initialised")
	}

	var (
		geoJSON  []byte
		metaJSON []byte
		err      error
	)

	if len(entry.GeoContext) > 0 {
		geoJSON, err = json.Marshal(entry.GeoContext)
		if err != nil {
			return fmt.Errorf("marshal geo context: %w", err)
		}
	}
	if len(entry.Metadata) > 0 {
		metaJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	const query = `
		INSERT INTO timeline_entries (
			timeline_id,
			user_id,
			category,
			confidence,
			geo_context,
			source_event_ids
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (timeline_id)
		DO UPDATE SET
			category = EXCLUDED.category,
			confidence = EXCLUDED.confidence,
			geo_context = EXCLUDED.geo_context,
			source_event_ids = EXCLUDED.source_event_ids
	`

	_, err = r.pool.Exec(
		ctx,
		query,
		entry.EventID,
		entry.UserID,
		entry.Category,
		entry.Confidence,
		geoJSON,
		entry.SourceEvents,
	)
	if err != nil {
		return fmt.Errorf("upsert timeline entry: %w", err)
	}

	return nil
}

// WithTx는 트랜잭션을 지원하기 위한 헬퍼(필요 시 사용)입니다.
func (r *Repository) WithTx(ctx context.Context, fn func(pgxtype pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
