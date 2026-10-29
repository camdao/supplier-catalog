package service

import (
	"catalog/global"
	"catalog/internal/model"
	"catalog/internal/repo"
	"fmt"
	"strings"
	"time"
)

type ProductService struct {
	Repo *repo.ProductRepo
}

func NewProductService() *ProductService {
	return &ProductService{
		Repo: repo.NewProductRepo(),
	}
}

// Query methods
func (s *ProductService) ListProducts(category string, includeWithdrawn bool) ([]model.Product, error) {
	return s.Repo.ListProducts(category, includeWithdrawn)
}

func (s *ProductService) GetProduct(id string) (*model.Product, error) {
	return s.Repo.GetProduct(id)
}

// Business logic types and functions
type ProductDecision struct {
	Action     string // "insert", "update", "skip"
	Product    model.Product
	ChangeType string // "added", "updated", "withdrawn", "skipped_older"
	Details    string
	OldVersion int64
	NewVersion int64
}

func ProcessItemDecision(item model.BatchItem, existing *model.Product) ProductDecision {
	decision := ProductDecision{
		Product: model.Product{
			ID:          item.ProductID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Category:    item.Category,
			Version:     item.Version,
			Withdrawn:   item.Withdrawn,
		},
		NewVersion: item.Version,
	}

	if existing != nil {
		decision.OldVersion = existing.Version
	}

	if existing == nil {
		decision.Action = "insert"
		if item.Withdrawn {
			decision.ChangeType = "withdrawn"
			decision.Details = fmt.Sprintf("new product added as withdrawn; name: %q", item.Name)
		} else {
			decision.ChangeType = "added"
			decision.Details = fmt.Sprintf("new product; name: %q", item.Name)
		}
		return decision
	}

	if existing.Version >= item.Version {
		decision.Action = "skip"
		decision.ChangeType = "skipped_older"
		decision.Details = fmt.Sprintf("existing version %d >= incoming %d", existing.Version, item.Version)
		return decision
	}

	decision.Action = "update"
	details := buildChangeDetails(existing, &item)

	if item.Withdrawn && !existing.Withdrawn {
		decision.ChangeType = "withdrawn"
	} else {
		decision.ChangeType = "updated"
	}

	if len(details) == 0 {
		decision.Details = fmt.Sprintf("no fields changed; version: %d → %d", existing.Version, item.Version)
	} else {
		details = append(details, fmt.Sprintf("version: %d → %d", existing.Version, item.Version))
		decision.Details = strings.Join(details, "; ")
	}

	return decision
}

func buildChangeDetails(existing *model.Product, item *model.BatchItem) []string {
	var details []string

	if existing.Name != item.Name {
		details = append(details, fmt.Sprintf("name: %q → %q", existing.Name, item.Name))
	}

	if existing.Description != item.Description {
		details = append(details, fmt.Sprintf("description: %q → %q", existing.Description, item.Description))
	}

	if existing.Price != item.Price {
		details = append(details, fmt.Sprintf("price: %d → %d", existing.Price, item.Price))
	}

	if existing.Category != item.Category {
		details = append(details, fmt.Sprintf("category: %q → %q", existing.Category, item.Category))
	}

	if existing.Withdrawn != item.Withdrawn {
		details = append(details, fmt.Sprintf("withdrawn: %t → %t", existing.Withdrawn, item.Withdrawn))
	}

	return details
}

// Batch processing
func ProcessBatchItems(
	updateID, batchID string,
	items []model.BatchItem,
) ([]model.ChangeRecord, error) {
	tx, err := global.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	productIDs := make([]string, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	productRepo := repo.NewProductRepo()
	existingProducts, err := productRepo.GetProductsByIDs(tx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("load existing products: %w", err)
	}

	now := time.Now()
	var productsToInsert []model.Product
	var productsToUpdate []model.Product
	changes := make([]model.ChangeRecord, 0, len(items))

	for _, item := range items {
		existing := existingProducts[item.ProductID]
		decision := ProcessItemDecision(item, existing)

		change := model.ChangeRecord{
			UpdateID:   updateID,
			BatchID:    batchID,
			ProductID:  item.ProductID,
			ChangeType: decision.ChangeType,
			OldVersion: decision.OldVersion,
			NewVersion: decision.NewVersion,
			Details:    decision.Details,
			CreatedAt:  now,
		}
		changes = append(changes, change)

		switch decision.Action {
		case "insert":
			productsToInsert = append(productsToInsert, decision.Product)
		case "update":
			productsToUpdate = append(productsToUpdate, decision.Product)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	updateRepo := repo.NewUpdateRepo()
	if err := updateRepo.BeginBatch(updateID, batchID, productsToInsert, productsToUpdate, changes); err != nil {
		return nil, fmt.Errorf("apply batch: %w", err)
	}

	return changes, nil
}
