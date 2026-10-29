package service

import (
	"catalog/internal/model"
	"testing"
)

func TestProcessItemDecision_Idempotency(t *testing.T) {
	item := model.BatchItem{
		ProductID:   "PROD-001",
		Name:        "Wireless Mouse",
		Description: "Ergonomic wireless mouse",
		Price:       2999,
		Category:    "Electronics",
		Version:     1,
		Withdrawn:   false,
	}

	existing := &model.Product{
		ID:          "PROD-001",
		Name:        "Wireless Mouse",
		Description: "Ergonomic wireless mouse",
		Price:       2999,
		Category:    "Electronics",
		Version:     1,
		Withdrawn:   false,
	}

	decision := ProcessItemDecision(item, existing)

	if decision.Action != "skip" {
		t.Errorf("expected action 'skip', got %s", decision.Action)
	}

	if decision.ChangeType != "skipped_older" {
		t.Errorf("expected change_type 'skipped_older', got %s", decision.ChangeType)
	}
}

func TestProcessItemDecision_VersionOrdering(t *testing.T) {
	itemV1 := model.BatchItem{
		ProductID:   "PROD-001",
		Name:        "Wireless Mouse",
		Description: "Old description",
		Price:       2999,
		Category:    "Electronics",
		Version:     1,
		Withdrawn:   false,
	}

	existingV2 := &model.Product{
		ID:          "PROD-001",
		Name:        "Wireless Mouse",
		Description: "Ergonomic wireless mouse",
		Price:       2499,
		Category:    "Electronics",
		Version:     2,
		Withdrawn:   false,
	}

	decision := ProcessItemDecision(itemV1, existingV2)

	if decision.Action != "skip" {
		t.Errorf("expected action 'skip', got %s", decision.Action)
	}

	if decision.ChangeType != "skipped_older" {
		t.Errorf("expected change_type 'skipped_older', got %s", decision.ChangeType)
	}

	if decision.OldVersion != 2 {
		t.Errorf("expected old version 2, got %d", decision.OldVersion)
	}

	if decision.NewVersion != 1 {
		t.Errorf("expected new version 1, got %d", decision.NewVersion)
	}
}

func TestProcessItemDecision_WithdrawalProtection(t *testing.T) {
	itemV2Active := model.BatchItem{
		ProductID:   "PROD-004",
		Name:        "Mechanical Keyboard",
		Description: "Cherry MX Brown mechanical keyboard",
		Price:       7999,
		Category:    "Electronics",
		Version:     2,
		Withdrawn:   false,
	}

	existingV3Withdrawn := &model.Product{
		ID:          "PROD-004",
		Name:        "Mechanical Keyboard",
		Description: "Cherry MX Brown mechanical keyboard",
		Price:       8999,
		Category:    "Electronics",
		Version:     3,
		Withdrawn:   true,
	}

	decision := ProcessItemDecision(itemV2Active, existingV3Withdrawn)

	if decision.Action != "skip" {
		t.Errorf("expected action 'skip', got %s", decision.Action)
	}

	if decision.ChangeType != "skipped_older" {
		t.Errorf("expected change_type 'skipped_older', got %s", decision.ChangeType)
	}
}

func TestProcessItemDecision_LegitimateRestore(t *testing.T) {
	itemV3Active := model.BatchItem{
		ProductID:   "PROD-009",
		Name:        "Cotton T-Shirt V2",
		Description: "Premium cotton crew neck, new design",
		Price:       1999,
		Category:    "Clothing",
		Version:     3,
		Withdrawn:   false,
	}

	existingV2Withdrawn := &model.Product{
		ID:          "PROD-009",
		Name:        "Cotton T-Shirt",
		Description: "Plain cotton crew neck",
		Price:       1499,
		Category:    "Clothing",
		Version:     2,
		Withdrawn:   true,
	}

	decision := ProcessItemDecision(itemV3Active, existingV2Withdrawn)

	if decision.Action != "update" {
		t.Errorf("expected action 'update', got %s", decision.Action)
	}

	if decision.ChangeType != "updated" {
		t.Errorf("expected change_type 'updated', got %s", decision.ChangeType)
	}

	if decision.Product.Withdrawn {
		t.Error("expected product to be restored (not withdrawn)")
	}

	if decision.Product.Version != 3 {
		t.Errorf("expected version 3, got %d", decision.Product.Version)
	}
}

func TestProcessItemDecision_NewProduct(t *testing.T) {
	item := model.BatchItem{
		ProductID:   "PROD-NEW",
		Name:        "New Product",
		Description: "Brand new product",
		Price:       5000,
		Category:    "Electronics",
		Version:     1,
		Withdrawn:   false,
	}

	decision := ProcessItemDecision(item, nil)

	if decision.Action != "insert" {
		t.Errorf("expected action 'insert', got %s", decision.Action)
	}

	if decision.ChangeType != "added" {
		t.Errorf("expected change_type 'added', got %s", decision.ChangeType)
	}

	if decision.OldVersion != 0 {
		t.Errorf("expected old version 0, got %d", decision.OldVersion)
	}
}

func TestProcessItemDecision_NewProductWithdrawn(t *testing.T) {
	item := model.BatchItem{
		ProductID:   "PROD-NEW",
		Name:        "Discontinued Product",
		Description: "Product added as withdrawn",
		Price:       5000,
		Category:    "Electronics",
		Version:     1,
		Withdrawn:   true,
	}

	decision := ProcessItemDecision(item, nil)

	if decision.Action != "insert" {
		t.Errorf("expected action 'insert', got %s", decision.Action)
	}

	if decision.ChangeType != "withdrawn" {
		t.Errorf("expected change_type 'withdrawn', got %s", decision.ChangeType)
	}

	if !decision.Product.Withdrawn {
		t.Error("expected product to be withdrawn")
	}
}

func TestProcessItemDecision_PriceUpdate(t *testing.T) {
	item := model.BatchItem{
		ProductID:   "PROD-001",
		Name:        "Wireless Mouse",
		Description: "Ergonomic wireless mouse",
		Price:       2499,
		Category:    "Electronics",
		Version:     2,
		Withdrawn:   false,
	}

	existing := &model.Product{
		ID:          "PROD-001",
		Name:        "Wireless Mouse",
		Description: "Ergonomic wireless mouse",
		Price:       2999,
		Category:    "Electronics",
		Version:     1,
		Withdrawn:   false,
	}

	decision := ProcessItemDecision(item, existing)

	if decision.Action != "update" {
		t.Errorf("expected action 'update', got %s", decision.Action)
	}

	if decision.ChangeType != "updated" {
		t.Errorf("expected change_type 'updated', got %s", decision.ChangeType)
	}

	if decision.Product.Price != 2499 {
		t.Errorf("expected price 2499, got %d", decision.Product.Price)
	}
}

func TestProcessItemDecision_Withdrawal(t *testing.T) {
	item := model.BatchItem{
		ProductID:   "PROD-004",
		Name:        "Mechanical Keyboard",
		Description: "Cherry MX Brown mechanical keyboard",
		Price:       8999,
		Category:    "Electronics",
		Version:     3,
		Withdrawn:   true,
	}

	existing := &model.Product{
		ID:          "PROD-004",
		Name:        "Mechanical Keyboard",
		Description: "Cherry MX Brown mechanical keyboard",
		Price:       8999,
		Category:    "Electronics",
		Version:     2,
		Withdrawn:   false,
	}

	decision := ProcessItemDecision(item, existing)

	if decision.Action != "update" {
		t.Errorf("expected action 'update', got %s", decision.Action)
	}

	if decision.ChangeType != "withdrawn" {
		t.Errorf("expected change_type 'withdrawn', got %s", decision.ChangeType)
	}

	if !decision.Product.Withdrawn {
		t.Error("expected product to be withdrawn")
	}
}
