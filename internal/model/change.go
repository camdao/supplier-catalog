package model

import "time"

type ChangeRecord struct {
	ID         int64
	UpdateID   string
	BatchID    string
	ProductID  string
	ChangeType string // "added", "updated", "withdrawn", "skipped_older", "skipped_duplicate"
	OldVersion int64
	NewVersion int64
	Details    string
	CreatedAt  time.Time
}
