package service

import (
	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repo"
	"catalog/thirt_part/supplier"
	dto_fake "catalog/thirt_part/supplier/dto"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type UpdateService struct {
	Supplier supplier.Supplier
	Repo     *repo.UpdateRepostruct
}

func NewUpdateService(sup supplier.Supplier) *UpdateService {
	return &UpdateService{
		Supplier: sup,
		Repo:     repo.NewUpdateRepo(),
	}
}

func (s *UpdateService) GetInProgressUpdate() (*model.Update, error) {
	return s.Repo.GetInProgressUpdate()
}

func (s *UpdateService) GetLatestUpdate() (*model.Update, error) {
	return s.Repo.GetLatestUpdate()
}

func (s *UpdateService) ListBatches(updateID string) ([]model.Batch, error) {
	return s.Repo.ListBatches(updateID)
}
func (e *UpdateService) StartUpdate() (*dto.UpdateResult, error) {
	if err := e.checkInProgressUpdate(); err != nil {
		return nil, err
	}

	updateID := generateIDFake()

	batchIDs, err := e.fetchBatchManifest(updateID)
	if err != nil {
		return nil, err
	}

	if err := e.createUpdateWithBatches(updateID, batchIDs); err != nil {
		return nil, err
	}

	return e.processBatches(updateID, batchIDs)
}

func (e *UpdateService) checkInProgressUpdate() error {
	exists, err := e.Repo.HasInProgressUpdate()
	if err != nil {
		return fmt.Errorf("check existing update: %w", err)
	}

	if exists {
		return fmt.Errorf("an update is already in progress; use 'update resume'")
	}

	return nil
}
func (e *UpdateService) fetchBatchManifest(
	updateID string,
) ([]string, error) {
	batchIDs, err := e.Supplier.FetchBatchManifest(updateID)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}

	return batchIDs, nil
}

func (e *UpdateService) createUpdateWithBatches(
	updateID string,
	batchIDs []string,
) error {
	update := &model.Update{
		ID:           updateID,
		Status:       model.UpdateInProgress,
		TotalBatches: len(batchIDs),
		StartedAt:    time.Now(),
	}

	batches := make([]*model.Batch, 0, len(batchIDs))

	for _, batchID := range batchIDs {
		batches = append(batches, &model.Batch{
			ID:       batchID,
			UpdateID: updateID,
			Status:   model.BatchPending,
		})
	}

	if err := e.Repo.CreateUpdateWithBatches(update, batches); err != nil {
		return fmt.Errorf("create update with batches: %w", err)
	}

	return nil
}

func generateIDFake() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *UpdateService) processBatches(
	updateID string,
	batchIDs []string,
) (*dto.UpdateResult, error) {
	result := &dto.UpdateResult{
		UpdateID:     updateID,
		TotalBatches: len(batchIDs),
	}

	allCompleted := true

	for _, batchID := range batchIDs {
		changes, skipped, err := e.processSingleBatch(updateID, batchID)
		if err != nil {
			return nil, err
		}

		if skipped {
			result.BatchesSkipped++
		}

		if changes == nil {
			allCompleted = false
			continue
		}

		result.BatchesCompleted++
		e.accumulateResult(result, changes)
		e.Repo.CompleteBatch(updateID, batchID)
	}

	status, err := e.finalizeUpdate(updateID, allCompleted)
	if err != nil {
		return nil, err
	}

	result.Status = status
	return result, nil
}

func (e *UpdateService) processSingleBatch(
	updateID string,
	batchID string,
) ([]model.ChangeRecord, bool, error) {
	batch, err := e.Repo.GetBatch(updateID, batchID)
	if err != nil {
		return nil, false, fmt.Errorf("get batch %s: %w", batchID, err)
	}

	if batch != nil && batch.Status == model.BatchCompleted {
		return nil, true, nil
	}

	items, err := e.Supplier.FetchBatch(batchID)
	if err != nil {
		if fErr := e.Repo.FailBatch(updateID, batchID, err.Error()); fErr != nil {
			return nil, false, fmt.Errorf("fail batch %s: %w", batchID, fErr)
		}
		return nil, false, nil
	}

	changes, err := e.Repo.BeginBatch(
		updateID,
		batchID,
		toModelItems(items),
	)
	if err != nil {
		if fErr := e.Repo.FailBatch(updateID, batchID, err.Error()); fErr != nil {
			return nil, false, fmt.Errorf("fail batch %s: %w", batchID, fErr)
		}
		return nil, false, nil
	}

	return changes, false, nil
}

func (e *UpdateService) accumulateResult(
	result *dto.UpdateResult,
	changes []model.ChangeRecord,
) {
	for _, change := range changes {
		switch change.ChangeType {
		case "added":
			result.ProductsAdded++
		case "updated":
			result.ProductsUpdated++
		case "withdrawn":
			result.ProductsWithdrawn++
		case "skipped_older":
			result.ProductsSkippedOlder++
		}
	}
}

func (e *UpdateService) finalizeUpdate(
	updateID string,
	allCompleted bool,
) (model.UpdateStatus, error) {
	if !allCompleted {
		return model.UpdateInProgress, nil
	}

	if err := e.Repo.CompleteUpdate(updateID); err != nil {
		return "", fmt.Errorf("complete update: %w", err)
	}

	return model.UpdateCompleted, nil
}

func toModelItems(items []dto_fake.BatchItem) []model.BatchItem {
	result := make([]model.BatchItem, 0, len(items))

	for _, item := range items {
		result = append(result, model.BatchItem{
			ProductID:   item.ProductID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Category:    item.Category,
			Version:     item.Version,
			Withdrawn:   item.Withdrawn,
		})
	}

	return result
}
