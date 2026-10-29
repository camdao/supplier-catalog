package repo

import (
	"catalog/global"
	"catalog/internal/model"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	var err error
	global.DB, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS updates (
			id           TEXT PRIMARY KEY,
			status       TEXT NOT NULL DEFAULT 'in_progress',
			total_batches INTEGER NOT NULL DEFAULT 0,
			started_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		);

		CREATE TABLE IF NOT EXISTS batches (
			id          TEXT NOT NULL,
			update_id   TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			error       TEXT NOT NULL DEFAULT '',
			processed_at DATETIME,
			PRIMARY KEY (update_id, id),
			FOREIGN KEY (update_id) REFERENCES updates(id)
		);

		CREATE TABLE IF NOT EXISTS products (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			price       INTEGER NOT NULL DEFAULT 0,
			category    TEXT NOT NULL DEFAULT '',
			version     INTEGER NOT NULL DEFAULT 0,
			withdrawn   INTEGER NOT NULL DEFAULT 0,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS changes (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			update_id   TEXT NOT NULL,
			batch_id    TEXT NOT NULL,
			product_id  TEXT NOT NULL,
			change_type TEXT NOT NULL,
			old_version INTEGER NOT NULL DEFAULT 0,
			new_version INTEGER NOT NULL DEFAULT 0,
			details     TEXT NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (update_id) REFERENCES updates(id)
		);
	`

	if _, err := global.DB.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
}

func teardownTestDB(t *testing.T) {
	t.Helper()
	if global.DB != nil {
		global.DB.Close()
	}
}

func createTestUpdate(t *testing.T, updateID string) {
	t.Helper()
	update := &model.Update{
		ID:           updateID,
		Status:       model.UpdateInProgress,
		TotalBatches: 1,
		StartedAt:    time.Now(),
	}
	batch := &model.Batch{
		ID:       "batch-1",
		UpdateID: updateID,
		Status:   model.BatchPending,
	}
	repo := NewUpdateRepo()
	if err := repo.CreateUpdateWithBatches(update, []*model.Batch{batch}); err != nil {
		t.Fatalf("failed to create test update: %v", err)
	}
}

func TestCRUD_GetProductsByIDs(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tx, _ := global.DB.Begin()
	defer tx.Rollback()

	now := time.Now()
	crud := NewProductRepo()

	products := []model.Product{
		{
			ID:          "PROD-001",
			Name:        "Product 1",
			Description: "Description 1",
			Price:       1000,
			Category:    "Category A",
			Version:     1,
			Withdrawn:   false,
		},
		{
			ID:          "PROD-002",
			Name:        "Product 2",
			Description: "Description 2",
			Price:       2000,
			Category:    "Category B",
			Version:     2,
			Withdrawn:   true,
		},
	}

	if err := crud.BulkInsertProducts(tx, products, now); err != nil {
		t.Fatalf("failed to insert products: %v", err)
	}
	tx.Commit()

	tx, _ = global.DB.Begin()
	defer tx.Rollback()

	loaded, err := crud.GetProductsByIDs(tx, []string{"PROD-001", "PROD-002", "PROD-999"})
	if err != nil {
		t.Fatalf("failed to get products: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 products, got %d", len(loaded))
	}

	if p := loaded["PROD-001"]; p == nil {
		t.Error("PROD-001 not found")
	} else {
		if p.Name != "Product 1" {
			t.Errorf("expected name 'Product 1', got %s", p.Name)
		}
		if p.Withdrawn {
			t.Error("expected PROD-001 not withdrawn")
		}
	}

	if p := loaded["PROD-002"]; p == nil {
		t.Error("PROD-002 not found")
	} else {
		if !p.Withdrawn {
			t.Error("expected PROD-002 withdrawn")
		}
	}

	if loaded["PROD-999"] != nil {
		t.Error("PROD-999 should not exist")
	}
}

func TestCRUD_BulkInsertProducts(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tx, _ := global.DB.Begin()
	defer tx.Rollback()

	now := time.Now()
	crud := NewProductRepo()

	products := []model.Product{
		{
			ID:          "PROD-001",
			Name:        "Product 1",
			Description: "Description 1",
			Price:       1000,
			Category:    "Category A",
			Version:     1,
			Withdrawn:   false,
		},
	}

	if err := crud.BulkInsertProducts(tx, products, now); err != nil {
		t.Fatalf("failed to insert products: %v", err)
	}
	tx.Commit()

	var count int
	err := global.DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count products: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 product, got %d", count)
	}
}

func TestCRUD_BulkUpdateProducts(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tx, _ := global.DB.Begin()
	now := time.Now()
	crud := NewProductRepo()

	products := []model.Product{
		{
			ID:          "PROD-001",
			Name:        "Product 1",
			Description: "Description 1",
			Price:       1000,
			Category:    "Category A",
			Version:     1,
			Withdrawn:   false,
		},
	}

	crud.BulkInsertProducts(tx, products, now)
	tx.Commit()

	tx, _ = global.DB.Begin()
	defer tx.Rollback()

	updatedProducts := []model.Product{
		{
			ID:          "PROD-001",
			Name:        "Updated Product 1",
			Description: "Updated Description",
			Price:       1500,
			Category:    "Category B",
			Version:     2,
			Withdrawn:   true,
		},
	}

	if err := crud.BulkUpdateProducts(tx, updatedProducts, now); err != nil {
		t.Fatalf("failed to update products: %v", err)
	}
	tx.Commit()

	var name string
	var price int64
	var version int64
	var withdrawn int
	err := global.DB.QueryRow("SELECT name, price, version, withdrawn FROM products WHERE id = ?", "PROD-001").
		Scan(&name, &price, &version, &withdrawn)
	if err != nil {
		t.Fatalf("failed to query product: %v", err)
	}

	if name != "Updated Product 1" {
		t.Errorf("expected name 'Updated Product 1', got %s", name)
	}
	if price != 1500 {
		t.Errorf("expected price 1500, got %d", price)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}
	if withdrawn != 1 {
		t.Errorf("expected withdrawn 1, got %d", withdrawn)
	}
}

func TestBeginBatch_Integration(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)
	createTestUpdate(t, "update-1")

	productsToInsert := []model.Product{
		{
			ID:          "PROD-001",
			Name:        "Product 1",
			Description: "Description 1",
			Price:       1000,
			Category:    "Category A",
			Version:     1,
			Withdrawn:   false,
		},
	}

	changes := []model.ChangeRecord{
		{
			UpdateID:   "update-1",
			BatchID:    "batch-1",
			ProductID:  "PROD-001",
			ChangeType: "added",
			OldVersion: 0,
			NewVersion: 1,
			Details:    "new product",
			CreatedAt:  time.Now(),
		},
	}

	repo := NewUpdateRepo()
	err := repo.BeginBatch("update-1", "batch-1", productsToInsert, nil, changes)
	if err != nil {
		t.Fatalf("BeginBatch failed: %v", err)
	}

	var productCount int
	global.DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&productCount)
	if productCount != 1 {
		t.Errorf("expected 1 product, got %d", productCount)
	}

	var changeCount int
	global.DB.QueryRow("SELECT COUNT(*) FROM changes").Scan(&changeCount)
	if changeCount != 1 {
		t.Errorf("expected 1 change record, got %d", changeCount)
	}

	batch, _ := repo.GetBatch("update-1", "batch-1")
	if batch.Status != model.BatchCompleted {
		t.Errorf("expected batch status 'completed', got %s", batch.Status)
	}
}
