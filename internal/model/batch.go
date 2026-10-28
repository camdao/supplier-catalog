package model

import "time"

type BatchStatus string

const (
	BatchPending   BatchStatus = "pending"
	BatchCompleted BatchStatus = "completed"
	BatchFailed    BatchStatus = "failed"
)

type Batch struct {
	ID          string
	UpdateID    string
	Status      BatchStatus
	Error       string
	ProcessedAt *time.Time
}

type BatchItem struct {
	ProductID   string
	Name        string
	Description string
	Price       int64 // price in cents
	Category    string
	Version     int64
	Withdrawn   bool
}
