package dto

import "catalog/internal/model"

type UpdateResult struct {
	UpdateID             string
	Status               model.UpdateStatus
	TotalBatches         int
	BatchesCompleted     int
	BatchesSkipped       int // already completed from prior run
	ProductsAdded        int
	ProductsUpdated      int
	ProductsWithdrawn    int
	ProductsSkippedOlder int
	Errors               []BatchError
}

type BatchError struct {
	BatchID string
	Error   string
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
