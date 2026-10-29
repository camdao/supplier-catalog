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
	productsToInsert []model.Product,
	productsToUpdate []model.Product,
	changes []model.ChangeRecord,
) error {
	tx, err := global.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	crud := NewProductRepo()

	if len(productsToInsert) > 0 {
		if err := crud.BulkInsertProducts(tx, productsToInsert, now); err != nil {
			return fmt.Errorf("bulk insert products: %w", err)
		}
	}

	if len(productsToUpdate) > 0 {
		if err := crud.BulkUpdateProducts(tx, productsToUpdate, now); err != nil {
			return fmt.Errorf("bulk update products: %w", err)
		}
	}

	if len(changes) > 0 {
		if err := crud.InsertChangeRecords(tx, changes); err != nil {
			return fmt.Errorf("insert changes: %w", err)
		}
	}

	result, err := tx.Exec(`
        UPDATE batches
        SET status = 'completed', processed_at = ?
        WHERE update_id = ? AND id = ?
    `, now, updateID, batchID)
	if err != nil {
		return fmt.Errorf("complete batch: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("batch not found or not uniquely identified")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

