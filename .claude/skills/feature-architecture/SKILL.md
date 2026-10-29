---
name: feature-architecture
description: Guide for creating new features following cmd → service → repo pattern
---

# Feature Architecture Pattern

This skill documents the three-layer architecture pattern used in this catalog project for implementing new features.

## Architecture Layers

All features follow this pattern:

```
cmd/ (Command Layer)
  ↓
internal/service/ (Service Layer)
  ↓
internal/repo/ (Repository Layer)
  ↓
Database (via global.DB)
```

## Layer Responsibilities

### 1. Command Layer ([cmd/](cmd/))
- **Purpose**: CLI interface, user interaction, flag handling
- **Dependencies**: Service layer only
- **Framework**: Cobra commands
- **Rules**:
  - No direct database access
  - No business logic
  - Call service methods only
  - Handle user input/output

### 2. Service Layer ([internal/service/](internal/service/))
- **Purpose**: Business logic, orchestration, validation
- **Dependencies**: Repository layer, external APIs/suppliers
- **Rules**:
  - No direct database access (use repo)
  - No CLI concerns (flags, formatting)
  - Return domain models
  - Handle business validation

### 3. Repository Layer ([internal/repo/](internal/repo/))
- **Purpose**: Data access, queries, transactions
- **Dependencies**: Database (global.DB), models
- **Rules**:
  - No business logic
  - No external API calls
  - Pure data operations
  - Return models from [internal/model/](internal/model/)

## Implementation Steps

### Step 1: Create Repository
**File**: `internal/repo/<feature>.repo.go`

```go
package repo

import (
	"catalog/global"
	"catalog/internal/model"
	"database/sql"
)

type FeatureRepo struct{}

func NewFeatureRepo() *FeatureRepo {
	return &FeatureRepo{}
}

// Add data access methods here
func (r *FeatureRepo) GetSomething(id string) (*model.Something, error) {
	// Database query using global.DB
	var result model.Something
	err := global.DB.QueryRow(`
		SELECT id, name, value
		FROM table_name
		WHERE id = ?
	`, id).Scan(&result.ID, &result.Name, &result.Value)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}
```

### Step 2: Create Service
**File**: `internal/service/<feature>.service.go`

```go
package service

import (
	"catalog/internal/model"
	"catalog/internal/repo"
)

type FeatureService struct {
	Repo *repo.FeatureRepo
}

func NewFeatureService() *FeatureService {
	return &FeatureService{
		Repo: repo.NewFeatureRepo(),
	}
}

// Add business logic methods here
func (s *FeatureService) DoSomething(input string) (*model.Result, error) {
	// Validate input (business logic)
	if input == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}
	
	// Call repo methods
	data, err := s.Repo.GetSomething(input)
	if err != nil {
		return nil, fmt.Errorf("fetch data: %w", err)
	}
	
	// Process data (business logic)
	result := &model.Result{
		Value: data.Value,
	}
	
	return result, nil
}
```

### Step 3: Create Command
**File**: `cmd/<feature>.go`

```go
package cmd

import (
	"catalog/internal/service"
	"fmt"
	
	"github.com/spf13/cobra"
)

var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Feature description",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.NewFeatureService()
		
		result, err := svc.DoSomething("input")
		if err != nil {
			return err
		}
		
		// Format output for user
		fmt.Printf("Result: %v\n", result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(featureCmd)
}
```

## Real Example: Changes Feature

### Repository ([internal/repo/changes.repo.go](internal/repo/changes.repo.go))
```go
type ChangesRepo struct{}

func NewChangesRepo() *ChangesRepo {
	return &ChangesRepo{}
}

func (r *ChangesRepo) GetLatestUpdate() (*model.Update, error) {
	var u model.Update
	var completedAt sql.NullTime
	err := global.DB.QueryRow(`
		SELECT id, status, total_batches, started_at, completed_at
		FROM updates
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&u.ID, &u.Status, &u.TotalBatches, &u.StartedAt, &completedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		u.CompletedAt = &completedAt.Time
	}
	return &u, nil
}

func (r *ChangesRepo) ListChanges(updateID string) ([]model.ChangeRecord, error) {
	rows, err := global.DB.Query(`
		SELECT id, update_id, batch_id, product_id, change_type,
		       old_version, new_version, details, created_at
		FROM changes
		WHERE update_id = ?
		ORDER BY created_at ASC
	`, updateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var changes []model.ChangeRecord
	for rows.Next() {
		var c model.ChangeRecord
		err := rows.Scan(&c.ID, &c.UpdateID, &c.BatchID, &c.ProductID,
			&c.ChangeType, &c.OldVersion, &c.NewVersion, &c.Details, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, rows.Err()
}
```

### Service ([internal/service/changes.service.go](internal/service/changes.service.go))
```go
type ChangesService struct {
	Repo *repo.ChangesRepo
}

func NewChangesService() *ChangesService {
	return &ChangesService{
		Repo: repo.NewChangesRepo(),
	}
}

func (s *ChangesService) GetLatestUpdate() (*model.Update, error) {
	return s.Repo.GetLatestUpdate()
}

func (s *ChangesService) ListChanges(updateID string) ([]model.ChangeRecord, error) {
	return s.Repo.ListChanges(updateID)
}
```

### Command ([cmd/changes.go](cmd/changes.go))
```go
var changesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List changes from a catalog update",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.NewChangesService()
		
		// Get latest update if no ID specified
		uid := changesUpdateID
		if uid == "" {
			latest, err := svc.GetLatestUpdate()
			if err != nil {
				return err
			}
			if latest == nil {
				fmt.Println("No updates found.")
				return nil
			}
			uid = latest.ID
			fmt.Printf("Showing changes for latest update: %s\n\n", uid)
		}
		
		// Fetch changes
		changes, err := svc.ListChanges(uid)
		if err != nil {
			return err
		}
		
		// Format and display results
		displayChangesTable(changes)
		return nil
	},
}
```

## Key Principles

1. **Dependency Direction**: Always flows downward (cmd → service → repo → db)
2. **Service Constructor**: Always creates its own repo instance via `New*Repo()`
3. **No Shortcuts**: Commands never call repos directly
4. **Model Passing**: Use domain models ([internal/model/](internal/model/)) across all layers
5. **Error Handling**: Wrap errors with context as they propagate up

## When to Add Logic at Each Layer

### Command Layer
- Flag parsing and validation
- Output formatting (tables, JSON, etc.)
- User confirmation prompts
- Progress indicators

### Service Layer
- Business rules and validation
- Multi-step orchestration
- Calling multiple repos
- External API integration
- Transaction coordination

### Repository Layer
- SQL queries
- Database transactions
- Batch operations
- Data mapping (DB ↔ models)

## Testing Strategy

- **Repo**: Test database queries with test database or mocks
- **Service**: Test business logic with mocked repo
- **Command**: Integration tests or manual CLI testing

## Anti-Patterns to Avoid

❌ Command calling repo directly (bypasses business logic)
❌ Service accessing `global.DB` directly (bypasses repo abstraction)
❌ Repo containing business logic (belongs in service)
❌ Mixing layers in one file
❌ Service methods that only pass through to repo (add business logic or merge into command)
❌ Creating helper functions like `openStore()` (use service constructors instead)

## Migration Guide

If you find code that doesn't follow this pattern:

1. **Direct DB access in commands**: Create repo → create service → update command
2. **Business logic in repos**: Move to service layer
3. **Helper functions returning repos**: Replace with service constructors
