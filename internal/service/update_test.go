package service

import (
	"catalog/global"
	"catalog/internal/model"
	"catalog/internal/repo"
	dto_fake "catalog/third_part/supplier/dto"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type mockSupplier struct {
	batchManifestFunc func(updateID string) ([]string, error)
	fetchBatchFunc    func(batchID string) ([]dto_fake.BatchItem, error)
}

func (m *mockSupplier) FetchBatchManifest(updateID string) ([]string, error) {
	if m.batchManifestFunc != nil {
		return m.batchManifestFunc(updateID)
	}
	return []string{"batch-1", "batch-2"}, nil
}

func (m *mockSupplier) FetchBatch(batchID string) ([]dto_fake.BatchItem, error) {
	if m.fetchBatchFunc != nil {
		return m.fetchBatchFunc(batchID)
	}
	return []dto_fake.BatchItem{
		{
			ProductID:   "PROD-001",
			Name:        "Test Product",
			Description: "Test Description",
			Price:       1000,
			Category:    "Test",
			Version:     1,
			Withdrawn:   false,
		},
	}, nil
}

func setupServiceTestDB(t *testing.T) {
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

func teardownServiceTestDB(t *testing.T) {
	t.Helper()
	if global.DB != nil {
		global.DB.Close()
	}
}

func TestStartUpdate_AllBatchesComplete(t *testing.T) {
	setupServiceTestDB(t)
	defer teardownServiceTestDB(t)

	mock := &mockSupplier{
		batchManifestFunc: func(updateID string) ([]string, error) {
			return []string{"batch-1", "batch-2"}, nil
		},
		fetchBatchFunc: func(batchID string) ([]dto_fake.BatchItem, error) {
			return []dto_fake.BatchItem{
				{
					ProductID:   "PROD-" + batchID,
					Name:        "Product from " + batchID,
					Description: "Test Description",
					Price:       1000,
					Category:    "Test",
					Version:     1,
					Withdrawn:   false,
				},
			}, nil
		},
	}

	service := NewUpdateService(mock)
	result, err := service.StartUpdate()

	if err != nil {
		t.Fatalf("StartUpdate failed: %v", err)
	}

	if result.Status != model.UpdateCompleted {
		t.Errorf("expected status 'completed', got %s", result.Status)
	}

	if result.TotalBatches != 2 {
		t.Errorf("expected 2 total batches, got %d", result.TotalBatches)
	}

	if result.BatchesCompleted != 2 {
		t.Errorf("expected 2 completed batches, got %d", result.BatchesCompleted)
	}

	if result.ProductsAdded != 2 {
		t.Errorf("expected 2 products added, got %d", result.ProductsAdded)
	}

	var count int
	err = global.DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count products: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 products in database, got %d", count)
	}
}

func TestStartUpdate_PartialCompletion(t *testing.T) {
	setupServiceTestDB(t)
	defer teardownServiceTestDB(t)

	batchCallCount := 0
	mock := &mockSupplier{
		batchManifestFunc: func(updateID string) ([]string, error) {
			return []string{"batch-1", "batch-2", "batch-3"}, nil
		},
		fetchBatchFunc: func(batchID string) ([]dto_fake.BatchItem, error) {
			batchCallCount++
			if batchID == "batch-2" {
				return nil, errors.New("supplier outage")
			}

			return []dto_fake.BatchItem{
				{
					ProductID:   "PROD-" + batchID,
					Name:        "Product from " + batchID,
					Description: "Test Description",
					Price:       1000,
					Category:    "Test",
					Version:     1,
					Withdrawn:   false,
				},
			}, nil
		},
	}

	service := NewUpdateService(mock)
	result, err := service.StartUpdate()

	if err != nil {
		t.Fatalf("StartUpdate failed: %v", err)
	}

	if result.Status != model.UpdateInProgress {
		t.Errorf("expected status 'in_progress', got %s", result.Status)
	}

	if result.TotalBatches != 3 {
		t.Errorf("expected 3 total batches, got %d", result.TotalBatches)
	}

	if result.BatchesCompleted != 2 {
		t.Errorf("expected 2 completed batches, got %d", result.BatchesCompleted)
	}

	if result.ProductsAdded != 2 {
		t.Errorf("expected 2 products added, got %d", result.ProductsAdded)
	}

	updateRepo := repo.NewUpdateRepo()
	update, err := updateRepo.GetInProgressUpdate()
	if err != nil {
		t.Fatalf("failed to get in-progress update: %v", err)
	}

	if update == nil {
		t.Fatal("expected in-progress update to exist")
	}

	batch, err := updateRepo.GetBatch(update.ID, "batch-2")
	if err != nil {
		t.Fatalf("failed to get batch: %v", err)
	}

	if batch.Status != model.BatchFailed {
		t.Errorf("expected batch-2 status 'failed', got %s", batch.Status)
	}

	if batch.Error == "" {
		t.Error("expected batch-2 to have error message")
	}
}

func TestResumeUpdate(t *testing.T) {
	setupServiceTestDB(t)
	defer teardownServiceTestDB(t)

	updateRepo := repo.NewUpdateRepo()
	update := &model.Update{
		ID:           "update-resume-test",
		Status:       model.UpdateInProgress,
		TotalBatches: 3,
		StartedAt:    time.Now(),
	}

	batches := []*model.Batch{
		{ID: "batch-1", UpdateID: "update-resume-test", Status: model.BatchCompleted},
		{ID: "batch-2", UpdateID: "update-resume-test", Status: model.BatchFailed},
		{ID: "batch-3", UpdateID: "update-resume-test", Status: model.BatchPending},
	}

	if err := updateRepo.CreateUpdateWithBatches(update, batches); err != nil {
		t.Fatalf("failed to create test update: %v", err)
	}

	now := time.Now()
	if _, err := global.DB.Exec("UPDATE batches SET status = 'completed', processed_at = ? WHERE update_id = ? AND id = ?",
		now, "update-resume-test", "batch-1"); err != nil {
		t.Fatalf("failed to mark batch-1 as completed: %v", err)
	}

	if _, err := global.DB.Exec("UPDATE batches SET status = 'failed', error = 'previous error', processed_at = ? WHERE update_id = ? AND id = ?",
		now, "update-resume-test", "batch-2"); err != nil {
		t.Fatalf("failed to mark batch-2 as failed: %v", err)
	}

	batch1CallCount := 0
	batch2CallCount := 0
	batch3CallCount := 0

	mock := &mockSupplier{
		fetchBatchFunc: func(batchID string) ([]dto_fake.BatchItem, error) {
			switch batchID {
			case "batch-1":
				batch1CallCount++
				return nil, errors.New("batch-1 should have been skipped")
			case "batch-2":
				batch2CallCount++
				return []dto_fake.BatchItem{
					{
						ProductID:   "PROD-002",
						Name:        "Product 2",
						Description: "Test",
						Price:       2000,
						Category:    "Test",
						Version:     1,
						Withdrawn:   false,
					},
				}, nil
			case "batch-3":
				batch3CallCount++
				return []dto_fake.BatchItem{
					{
						ProductID:   "PROD-003",
						Name:        "Product 3",
						Description: "Test",
						Price:       3000,
						Category:    "Test",
						Version:     1,
						Withdrawn:   false,
					},
				}, nil
			}
			return nil, errors.New("unexpected batch")
		},
	}

	service := NewUpdateService(mock)
	result, err := service.ResumeUpdate()

	if err != nil {
		t.Fatalf("ResumeUpdate failed: %v", err)
	}

	if batch1CallCount > 0 {
		t.Error("batch-1 should not have been fetched (already completed)")
	}

	if batch2CallCount != 1 {
		t.Errorf("expected batch-2 to be fetched once (retry), got %d", batch2CallCount)
	}

	if batch3CallCount != 1 {
		t.Errorf("expected batch-3 to be fetched once, got %d", batch3CallCount)
	}

	if result.Status != model.UpdateCompleted {
		t.Errorf("expected status 'completed', got %s", result.Status)
	}

	if result.BatchesCompleted != 2 {
		t.Errorf("expected 2 newly completed batches, got %d", result.BatchesCompleted)
	}

	if result.BatchesSkipped != 1 {
		t.Errorf("expected 1 skipped batch, got %d", result.BatchesSkipped)
	}

	if result.ProductsAdded != 2 {
		t.Errorf("expected 2 products added, got %d", result.ProductsAdded)
	}
}

func TestStartUpdate_PreventsConcurrentUpdates(t *testing.T) {
	setupServiceTestDB(t)
	defer teardownServiceTestDB(t)

	updateRepo := repo.NewUpdateRepo()
	existingUpdate := &model.Update{
		ID:           "existing-update",
		Status:       model.UpdateInProgress,
		TotalBatches: 1,
		StartedAt:    time.Now(),
	}
	batch := &model.Batch{
		ID:       "batch-1",
		UpdateID: "existing-update",
		Status:   model.BatchPending,
	}

	if err := updateRepo.CreateUpdateWithBatches(existingUpdate, []*model.Batch{batch}); err != nil {
		t.Fatalf("failed to create existing update: %v", err)
	}

	mock := &mockSupplier{}
	service := NewUpdateService(mock)

	_, err := service.StartUpdate()
	if err == nil {
		t.Error("expected error when starting update with existing in-progress update")
	}

	if err != nil && err.Error() != "an update is already in progress; use 'update resume'" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResumeUpdate_NoInProgressUpdate(t *testing.T) {
	setupServiceTestDB(t)
	defer teardownServiceTestDB(t)

	mock := &mockSupplier{}
	service := NewUpdateService(mock)

	_, err := service.ResumeUpdate()
	if err == nil {
		t.Error("expected error when no in-progress update exists")
	}

	if err != nil && err.Error() != "no update in progress to resume" {
		t.Errorf("unexpected error message: %v", err)
	}
}
