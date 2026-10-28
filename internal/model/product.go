package model

import "time"

type Product struct {
	ID          string
	Name        string
	Description string
	Price       int64 // price in cents
	Category    string
	Version     int64
	Withdrawn   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
