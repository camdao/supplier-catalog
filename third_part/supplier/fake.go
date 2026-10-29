package supplier

import (
	dto_fake "catalog/third_part/supplier/dto"
)

type FakeSupplier struct{}

func NewFakeSupplier() *FakeSupplier {
	return &FakeSupplier{}
}

func (f *FakeSupplier) FetchBatchManifest(updateID string) ([]string, error) {
	return []string{
		"batch-001",
		"batch-002",
		"batch-003",
		"batch-004",
		"batch-005",
	}, nil
}

func (f *FakeSupplier) FetchBatch(batchID string) ([]dto_fake.BatchItem, error) {
	return generateBatch(batchID), nil
}

type productTemplate struct {
	id       string
	name     string
	desc     string
	price    int64
	category string
}

func generateBatch(batchID string) []dto_fake.BatchItem {
	switch batchID {
	case "batch-001":
		return batch1()
	case "batch-002":
		return batch2()
	case "batch-003":
		return batch3()
	case "batch-004":
		return batch4()
	case "batch-005":
		return batch5()
	default:
		return nil
	}
}

// batch1: 10 new products (version 1)
func batch1() []dto_fake.BatchItem {
	products := []productTemplate{
		{"PROD-001", "Wireless Mouse", "Ergonomic wireless mouse with USB receiver", 2999, "Electronics"},
		{"PROD-002", "USB-C Cable", "Braided USB-C to USB-A cable, 2m", 1299, "Electronics"},
		{"PROD-003", "Go Programming Language", "The Go Programming Language by Donovan & Kernighan", 3499, "Books"},
		{"PROD-004", "Mechanical Keyboard", "Cherry MX Brown mechanical keyboard", 8999, "Electronics"},
		{"PROD-005", "Running Shoes", "Lightweight running shoes, breathable mesh", 6999, "Clothing"},
		{"PROD-006", "Organic Coffee Beans", "Single-origin Arabica, 1kg bag", 1899, "Food"},
		{"PROD-007", "Hex Key Set", "Metric hex key set, 1.5-10mm", 1599, "Tools"},
		{"PROD-008", "Laptop Stand", "Adjustable aluminum laptop stand", 4599, "Electronics"},
		{"PROD-009", "Cotton T-Shirt", "Plain cotton crew neck, assorted colors", 1499, "Clothing"},
		{"PROD-010", "Drill Bit Set", "HSS drill bits, 1-13mm, 25 pieces", 2499, "Tools"},
	}
	items := make([]dto_fake.BatchItem, len(products))
	for i, p := range products {
		items[i] = dto_fake.BatchItem{
			ProductID:   p.id,
			Name:        p.name,
			Description: p.desc,
			Price:       p.price,
			Category:    p.category,
			Version:     1,
			Withdrawn:   false,
		}
	}
	return items
}

// batch2: Updates to some existing products (version 2) + new products
func batch2() []dto_fake.BatchItem {
	return []dto_fake.BatchItem{
		// Price update for existing products
		{ProductID: "PROD-001", Name: "Wireless Mouse", Description: "Ergonomic wireless mouse with USB receiver", Price: 2499, Category: "Electronics", Version: 2, Withdrawn: false},
		{ProductID: "PROD-006", Name: "Organic Coffee Beans", Description: "Single-origin Arabica, 1kg bag - new harvest", Price: 2099, Category: "Food", Version: 2, Withdrawn: false},
		// New products
		{ProductID: "PROD-011", Name: "Bluetooth Speaker", Description: "Portable waterproof Bluetooth speaker", Price: 3999, Category: "Electronics", Version: 1, Withdrawn: false},
		{ProductID: "PROD-012", Name: "Design Patterns", Description: "Design Patterns by Gang of Four", Price: 4299, Category: "Books", Version: 1, Withdrawn: false},
		{ProductID: "PROD-013", Name: "Winter Jacket", Description: "Insulated winter jacket, waterproof", Price: 12999, Category: "Clothing", Version: 1, Withdrawn: false},
		{ProductID: "PROD-014", Name: "Green Tea", Description: "Japanese Sencha green tea, 200g", Price: 1299, Category: "Food", Version: 1, Withdrawn: false},
		{ProductID: "PROD-015", Name: "Tape Measure", Description: "5m retractable tape measure", Price: 899, Category: "Tools", Version: 1, Withdrawn: false},
		// Duplicate: PROD-003 at version 1 again (should be skipped — idempotency)
		{ProductID: "PROD-003", Name: "Go Programming Language", Description: "The Go Programming Language by Donovan & Kernighan", Price: 3499, Category: "Books", Version: 1, Withdrawn: false},
	}
}

// batch3: Withdrawals + out-of-order versions
func batch3() []dto_fake.BatchItem {
	return []dto_fake.BatchItem{
		// Withdraw PROD-004 (keyboard discontinued)
		{ProductID: "PROD-004", Name: "Mechanical Keyboard", Description: "Cherry MX Brown mechanical keyboard", Price: 8999, Category: "Electronics", Version: 3, Withdrawn: true},
		// Withdraw PROD-009 (t-shirt discontinued)
		{ProductID: "PROD-009", Name: "Cotton T-Shirt", Description: "Plain cotton crew neck, assorted colors", Price: 1499, Category: "Clothing", Version: 2, Withdrawn: true},
		// Out-of-order: PROD-001 version 1 arrives AFTER version 2 was applied (should be skipped)
		{ProductID: "PROD-001", Name: "Wireless Mouse", Description: "Old description", Price: 2999, Category: "Electronics", Version: 1, Withdrawn: false},
		// New products
		{ProductID: "PROD-016", Name: "Adjustable Wrench", Description: "10-inch adjustable wrench, chrome vanadium", Price: 1899, Category: "Tools", Version: 1, Withdrawn: false},
		{ProductID: "PROD-017", Name: "Trail Mix", Description: "Premium nut and dried fruit mix, 500g", Price: 899, Category: "Food", Version: 1, Withdrawn: false},
		{ProductID: "PROD-018", Name: "Clean Code", Description: "Clean Code by Robert C. Martin", Price: 3799, Category: "Books", Version: 1, Withdrawn: false},
	}
}

// batch4: More updates + attempt to un-withdraw with old version (should fail)
func batch4() []dto_fake.BatchItem {
	return []dto_fake.BatchItem{
		// Try to restore PROD-004 with version 2 (older than withdrawal at version 3 — should be skipped)
		{ProductID: "PROD-004", Name: "Mechanical Keyboard", Description: "Cherry MX Brown mechanical keyboard", Price: 7999, Category: "Electronics", Version: 2, Withdrawn: false},
		// Legitimately restore PROD-009 with version 3 (newer than withdrawal at version 2)
		{ProductID: "PROD-009", Name: "Cotton T-Shirt V2", Description: "Premium cotton crew neck, new design", Price: 1999, Category: "Clothing", Version: 3, Withdrawn: false},
		// New products
		{ProductID: "PROD-019", Name: "Webcam HD", Description: "1080p HD webcam with autofocus", Price: 5999, Category: "Electronics", Version: 1, Withdrawn: false},
		{ProductID: "PROD-020", Name: "Rain Boots", Description: "Waterproof rubber rain boots", Price: 3499, Category: "Clothing", Version: 1, Withdrawn: false},
		{ProductID: "PROD-021", Name: "Sourdough Bread Mix", Description: "Artisan sourdough bread flour mix, 1.5kg", Price: 699, Category: "Food", Version: 1, Withdrawn: false},
	}
}

// batch5: Final batch — price updates, new products, duplicate of earlier batch item
func batch5() []dto_fake.BatchItem {
	return []dto_fake.BatchItem{
		// Price updates
		{ProductID: "PROD-002", Name: "USB-C Cable", Description: "Braided USB-C to USB-A cable, 2m - improved strain relief", Price: 1099, Category: "Electronics", Version: 2, Withdrawn: false},
		{ProductID: "PROD-007", Name: "Hex Key Set", Description: "Metric hex key set, 1.5-10mm, with holder", Price: 1799, Category: "Tools", Version: 2, Withdrawn: false},
		// New products
		{ProductID: "PROD-022", Name: "Espresso Machine", Description: "Semi-automatic espresso machine, 15 bar", Price: 29999, Category: "Electronics", Version: 1, Withdrawn: false},
		{ProductID: "PROD-023", Name: "Refactoring", Description: "Refactoring by Martin Fowler, 2nd edition", Price: 4599, Category: "Books", Version: 1, Withdrawn: false},
		{ProductID: "PROD-024", Name: "Safety Glasses", Description: "Impact-resistant polycarbonate safety glasses", Price: 1299, Category: "Tools", Version: 1, Withdrawn: false},
		{ProductID: "PROD-025", Name: "Hiking Backpack", Description: "30L daypack with rain cover", Price: 5999, Category: "Clothing", Version: 1, Withdrawn: false},
		// Duplicate: PROD-011 version 1 again (already added in batch2 — should be skipped)
		{ProductID: "PROD-011", Name: "Bluetooth Speaker", Description: "Portable waterproof Bluetooth speaker", Price: 3999, Category: "Electronics", Version: 1, Withdrawn: false},
	}
}
