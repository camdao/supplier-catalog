package dto_fake

type BatchItem struct {
	ProductID   string
	Name        string
	Description string
	Price       int64 // price in cents
	Category    string
	Version     int64
	Withdrawn   bool
}
