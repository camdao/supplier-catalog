package repo

import (
	"catalog/global"
	"catalog/internal/model"
	"database/sql"
)

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
		err := rows.Scan(
			&c.ID,
			&c.UpdateID,
			&c.BatchID,
			&c.ProductID,
			&c.ChangeType,
			&c.OldVersion,
			&c.NewVersion,
			&c.Details,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return changes, nil
}
