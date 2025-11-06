package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event는 activity_events 테이블에 삽입될 데이터 모델입니다.
type Event struct {
	EventID       string
	UserID        string
	Source        string
	TimestampStart time.Time
	TimestampEnd   time.Time
	Metadata       map[string]any
}

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

// Save는 activity_events 테이블에 이벤트를 저장합니다.
func (r *EventRepository) Save(ctx context.Context, event Event) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("event repository not initialised")
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	const query = `
		INSERT INTO activity_events (
			event_id,
			user_id,
			source,
			timestamp_start,
			timestamp_end,
			metadata
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.pool.Exec(
		ctx,
		query,
		event.EventID,
		event.UserID,
		event.Source,
		event.TimestampStart,
		event.TimestampEnd,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("insert activity_event: %w", err)
	}

	return nil
}

// Ping은 데이터베이스 연결 상태를 확인합니다.
func (r *EventRepository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("event repository not initialised")
	}
	return r.pool.Ping(ctx)
}
