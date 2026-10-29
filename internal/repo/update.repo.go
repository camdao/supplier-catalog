package repo

import (
	"catalog/global"
	"catalog/internal/model"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type UpdateRepostruct struct{}

func NewUpdateRepo() *UpdateRepostruct {
	return &UpdateRepostruct{}
}

func (r *UpdateRepostruct) HasInProgressUpdate() (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1
            FROM updates
            WHERE status = ?
        )
    `

	var exists bool
	err := global.DB.QueryRow(query, "in_progress").Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *UpdateRepostruct) GetInProgressUpdate() (*model.Update, error) {
	var u model.Update
	var completedAt sql.NullTime
	err := global.DB.QueryRow(`
		SELECT id, status, total_batches, started_at, completed_at
		FROM updates
		WHERE status = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, "in_progress").Scan(&u.ID, &u.Status, &u.TotalBatches, &u.StartedAt, &completedAt)

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

func (r *UpdateRepostruct) GetLatestUpdate() (*model.Update, error) {
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

func (r *UpdateRepostruct) ListBatches(updateID string) ([]model.Batch, error) {
	rows, err := global.DB.Query(`
		SELECT id, update_id, status, error, processed_at
		FROM batches
		WHERE update_id = ?
		ORDER BY id ASC
	`, updateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []model.Batch
	for rows.Next() {
		var b model.Batch
		var processedAt sql.NullTime
		err := rows.Scan(&b.ID, &b.UpdateID, &b.Status, &b.Error, &processedAt)
		if err != nil {
			return nil, err
		}
		if processedAt.Valid {
			b.ProcessedAt = &processedAt.Time
		}
		batches = append(batches, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return batches, nil
}
func (s *UpdateRepostruct) CreateUpdateWithBatches(
	update *model.Update,
	batches []*model.Batch,
) error {
	tx, err := global.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO updates
         (id, status, total_batches, started_at)
         VALUES (?, ?, ?, ?)`,
		update.ID,
		update.Status,
		update.TotalBatches,
		update.StartedAt,
	)
	if err != nil {
		return fmt.Errorf("create update: %w", err)
	}

	if len(batches) > 0 {
		var query strings.Builder
		query.WriteString(
			`INSERT OR IGNORE INTO batches (id, update_id, status) VALUES `,
		)

		args := make([]any, 0, len(batches)*3)

		for i, batch := range batches {
			if i > 0 {
				query.WriteString(", ")
			}

			query.WriteString("(?, ?, ?)")
			args = append(args, batch.ID, batch.UpdateID, batch.Status)
		}

		if _, err := tx.Exec(query.String(), args...); err != nil {
			return fmt.Errorf("create batches: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
func (s *UpdateRepostruct) FailBatch(updateID, batchID string, errMsg string) error {
	_, err := global.DB.Exec(
		`UPDATE batches SET status = 'failed', error = ?, processed_at = ? WHERE update_id = ? AND id = ?`,
		errMsg, time.Now(), updateID, batchID,
	)
	return err
}

func (s *UpdateRepostruct) CompleteUpdate(id string) error {
	_, err := global.DB.Exec(
		`UPDATE updates SET status = 'completed', completed_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	return err
}

func (s *UpdateRepostruct) GetBatch(updateID, batchID string) (*model.Batch, error) {
	var b model.Batch
	var processedAt sql.NullTime
	err := global.DB.QueryRow(
		`SELECT id, update_id, status, error, processed_at FROM batches WHERE update_id = ? AND id = ?`,
		updateID, batchID,
	).Scan(&b.ID, &b.UpdateID, &b.Status, &b.Error, &processedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if processedAt.Valid {
		b.ProcessedAt = &processedAt.Time
	}
	return &b, nil
}

func (s *UpdateRepostruct) CompleteBatch(updateID, batchID string) error {
	_, err := global.DB.Exec(
		`UPDATE batches SET status = 'completed', processed_at = ? WHERE update_id = ? AND id = ?`,
		time.Now(), updateID, batchID,
	)
	return err
}

func (s *UpdateRepostruct) BeginBatch(
	updateID, batchID string,
	items []model.BatchItem,
) ([]model.ChangeRecord, error) {
	tx, err := global.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	existingProducts, err := s.loadExistingProducts(tx, items)
	if err != nil {
		return nil, err
	}

	changes := make([]model.ChangeRecord, 0, len(items))
	toInsert := make([]model.BatchItem, 0)
	toUpdate := make([]model.BatchItem, 0)

	for _, item := range items {
		existing, exists := existingProducts[item.ProductID]

		change := s.processItem(
			item,
			updateID,
			batchID,
			now,
			existing,
			exists,
		)

		changes = append(changes, *change)

		switch {
		case !exists:
			toInsert = append(toInsert, item)
		case exists && existing.version < item.Version:
			toUpdate = append(toUpdate, item)
		}
	}

	if len(toInsert) > 0 {
		if err := bulkInsertProducts(tx, toInsert, now); err != nil {
			return nil, fmt.Errorf("bulk insert products: %w", err)
		}
	}

	if len(toUpdate) > 0 {
		if err := bulkUpdateProducts(tx, toUpdate, now); err != nil {
			return nil, fmt.Errorf("bulk update products: %w", err)
		}
	}

	if len(changes) > 0 {
		if err := bulkInsertChanges(tx, changes); err != nil {
			return nil, fmt.Errorf("bulk insert changes: %w", err)
		}
	}

	result, err := tx.Exec(`
        UPDATE batches
        SET status = 'completed', processed_at = ?
        WHERE update_id = ? AND id = ?
    `, now, updateID, batchID)
	if err != nil {
		return nil, fmt.Errorf("complete batch: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, fmt.Errorf("batch not found or not uniquely identified")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return changes, nil
}

type existingProduct struct {
	version     int64
	withdrawn   bool
	name        string
	description string
	price       int64
	category    string
}

func (s *UpdateRepostruct) loadExistingProducts(
	tx *sql.Tx,
	items []model.BatchItem,
) (map[string]existingProduct, error) {
	products := make(map[string]existingProduct, len(items))

	const chunkSize = 500

	for start := 0; start < len(items); start += chunkSize {
		end := min(start+chunkSize, len(items))
		chunk := items[start:end]

		placeholders := make([]string, len(chunk))
		args := make([]any, 0, len(chunk))

		for i, item := range chunk {
			placeholders[i] = "?"
			args = append(args, item.ProductID)
		}

		query := `
            SELECT id, name, description, price,
                   category, version, withdrawn
            FROM products
            WHERE id IN (` + strings.Join(placeholders, ",") + `)
        `

		rows, err := tx.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("load existing products: %w", err)
		}

		for rows.Next() {
			var id string
			var p existingProduct
			var withdrawn int

			if err := rows.Scan(
				&id,
				&p.name,
				&p.description,
				&p.price,
				&p.category,
				&p.version,
				&withdrawn,
			); err != nil {
				rows.Close()
				return nil, err
			}

			p.withdrawn = withdrawn != 0
			products[id] = p
		}

		err = rows.Err()
		rows.Close()

		if err != nil {
			return nil, err
		}
	}

	return products, nil
}
func (s *UpdateRepostruct) processItem(
	item model.BatchItem,
	updateID, batchID string,
	now time.Time,
	existing existingProduct,
	exists bool,
) *model.ChangeRecord {
	change := &model.ChangeRecord{
		UpdateID:   updateID,
		BatchID:    batchID,
		ProductID:  item.ProductID,
		NewVersion: item.Version,
		CreatedAt:  now,
	}

	if exists {
		change.OldVersion = existing.version
	}

	switch {
	case exists && existing.version >= item.Version:
		change.ChangeType = "skipped_older"
		change.Details = fmt.Sprintf(
			"existing version %d >= incoming %d",
			existing.version,
			item.Version,
		)

	case !exists:
		if item.Withdrawn {
			change.ChangeType = "withdrawn"
			change.Details = fmt.Sprintf(
				"new product added as withdrawn; name: %q",
				item.Name,
			)
		} else {
			change.ChangeType = "added"
			change.Details = fmt.Sprintf("new product; name: %q", item.Name)
		}

	default:
		details := diffProduct(existing, item)

		if item.Withdrawn && !existing.withdrawn {
			change.ChangeType = "withdrawn"
		} else {
			change.ChangeType = "updated"
		}

		if len(details) == 0 {
			change.Details = fmt.Sprintf(
				"no fields changed; version: %d → %d",
				existing.version,
				item.Version,
			)
		} else {
			details = append(details, fmt.Sprintf(
				"version: %d → %d",
				existing.version,
				item.Version,
			))
			change.Details = strings.Join(details, "; ")
		}
	}

	return change
}

func bulkInsertProducts(
	tx *sql.Tx,
	items []model.BatchItem,
	now time.Time,
) error {
	if len(items) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`
		INSERT INTO products (
			id, name, description, price, category,
			version, withdrawn, created_at, updated_at
		) VALUES `)

	args := make([]any, 0, len(items)*9)
	for i, item := range items {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			item.ProductID,
			item.Name,
			item.Description,
			item.Price,
			item.Category,
			item.Version,
			boolToInt(item.Withdrawn),
			now,
			now,
		)
	}

	_, err := tx.Exec(query.String(), args...)
	return err
}

func bulkUpdateProducts(
	tx *sql.Tx,
	items []model.BatchItem,
	now time.Time,
) error {
	if len(items) == 0 {
		return nil
	}

	const chunkSize = 500
	for start := 0; start < len(items); start += chunkSize {
		end := min(start+chunkSize, len(items))
		chunk := items[start:end]

		for _, item := range chunk {
			_, err := tx.Exec(`
				UPDATE products
				SET name = ?,
					description = ?,
					price = ?,
					category = ?,
					version = ?,
					withdrawn = ?,
					updated_at = ?
				WHERE id = ?
			`,
				item.Name,
				item.Description,
				item.Price,
				item.Category,
				item.Version,
				boolToInt(item.Withdrawn),
				now,
				item.ProductID,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func bulkInsertChanges(
	tx *sql.Tx,
	changes []model.ChangeRecord,
) error {
	if len(changes) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`
		INSERT INTO changes (
			update_id, batch_id, product_id, change_type,
			old_version, new_version, details, created_at
		) VALUES `)

	args := make([]any, 0, len(changes)*8)
	for i, change := range changes {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("(?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			change.UpdateID,
			change.BatchID,
			change.ProductID,
			change.ChangeType,
			change.OldVersion,
			change.NewVersion,
			change.Details,
			change.CreatedAt,
		)
	}

	_, err := tx.Exec(query.String(), args...)
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func diffProduct(
	existing existingProduct,
	item model.BatchItem,
) []string {
	var details []string

	if existing.name != item.Name {
		details = append(details, fmt.Sprintf(
			"name: %q → %q", existing.name, item.Name,
		))
	}

	if existing.description != item.Description {
		details = append(details, fmt.Sprintf(
			"description: %q → %q",
			existing.description, item.Description,
		))
	}

	if existing.price != item.Price {
		details = append(details, fmt.Sprintf(
			"price: %d → %d", existing.price, item.Price,
		))
	}

	if existing.category != item.Category {
		details = append(details, fmt.Sprintf(
			"category: %q → %q",
			existing.category, item.Category,
		))
	}

	if existing.withdrawn != item.Withdrawn {
		details = append(details, fmt.Sprintf(
			"withdrawn: %t → %t",
			existing.withdrawn, item.Withdrawn,
		))
	}

	return details
}
