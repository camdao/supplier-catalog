package model

import "time"

type UpdateStatus string

const (
	UpdateInProgress UpdateStatus = "in_progress"
	UpdateCompleted  UpdateStatus = "completed"
	UpdateFailed     UpdateStatus = "failed"
)

type Update struct {
	ID           string
	Status       UpdateStatus
	TotalBatches int
	StartedAt    time.Time
	CompletedAt  *time.Time
}
