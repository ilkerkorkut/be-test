package entity

import (
	"encoding/json"
	"time"
)

type OutboxEvent struct {
	ID          string          `json:"id"`
	Topic       string          `json:"topic"`
	Key         string          `json:"key"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"createdAt"`
	PublishedAt *time.Time      `json:"publishedAt,omitempty"`
}
