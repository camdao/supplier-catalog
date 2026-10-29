package repo

import (
	"catalog/global"
	"catalog/internal/model"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ProductRepo struct{}

func NewProductRepo() *ProductRepo {
	return &ProductRepo{}
}

// Query operations
func (r *ProductRepo) ListProducts(category string, includeWithdrawn bool) ([]model.Product, error) {
	query := `
		SELECT id, name, description, price, category, version, withdrawn, created_at, updated_at
		FROM products
		WHERE 1=1
	`
	args := []interface{}{}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	if !includeWithdrawn {
		query += " AND withdrawn = 0"
	}

	query += " ORDER BY created_at DESC"

	rows, err := global.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		var withdrawn int
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Category,
			&p.Version,
			&withdrawn,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		p.Withdrawn = withdrawn != 0
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *ProductRepo) GetProduct(id string) (*model.Product, error) {
	var p model.Product
	var withdrawn int

	err := global.DB.QueryRow(`
		SELECT id, name, description, price, category, version, withdrawn, created_at, updated_at
		FROM products
		WHERE id = ?
	`, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.Category,
		&p.Version,
		&withdrawn,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.Withdrawn = withdrawn != 0
	return &p, nil
}

// CRUD operations for batch processing
func (r *ProductRepo) GetProductsByIDs(tx *sql.Tx, ids []string) (map[string]*model.Product, error) {
	if len(ids) == 0 {
		return make(map[string]*model.Product), nil
	}

	products := make(map[string]*model.Product)
	const chunkSize = 500

	for start := 0; start < len(ids); start += chunkSize {
		end := min(start+chunkSize, len(ids))
		chunk := ids[start:end]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}

		query := `
			SELECT id, name, description, price, category, version, withdrawn, created_at, updated_at
			FROM products
			WHERE id IN (` + strings.Join(placeholders, ",") + `)
		`

		var rows *sql.Rows
		var err error
		if tx != nil {
			rows, err = tx.Query(query, args...)
		} else {
			rows, err = global.DB.Query(query, args...)
		}

		if err != nil {
			return nil, fmt.Errorf("query products: %w", err)
		}

		for rows.Next() {
			var p model.Product
			var withdrawn int
			err := rows.Scan(
				&p.ID,
				&p.Name,
				&p.Description,
				&p.Price,
				&p.Category,
				&p.Version,
				&withdrawn,
				&p.CreatedAt,
				&p.UpdatedAt,
			)
			if err != nil {
				rows.Close()
				return nil, err
			}
			p.Withdrawn = withdrawn != 0
			products[p.ID] = &p
		}

		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}

	return products, nil
}

func (r *ProductRepo) BulkInsertProducts(tx *sql.Tx, products []model.Product, now time.Time) error {
	if len(products) == 0 {
		return nil
	}

	var query strings.Builder
	query.WriteString(`
		INSERT INTO products (
			id, name, description, price, category,
			version, withdrawn, created_at, updated_at
		) VALUES `)

	args := make([]any, 0, len(products)*9)
	for i, p := range products {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			p.ID,
			p.Name,
			p.Description,
			p.Price,
			p.Category,
			p.Version,
			boolToInt(p.Withdrawn),
			now,
			now,
		)
	}

	_, err := tx.Exec(query.String(), args...)
	return err
}

func (r *ProductRepo) BulkUpdateProducts(tx *sql.Tx, products []model.Product, now time.Time) error {
	if len(products) == 0 {
		return nil
	}

	for _, p := range products {
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
			p.Name,
			p.Description,
			p.Price,
			p.Category,
			p.Version,
			boolToInt(p.Withdrawn),
			now,
			p.ID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ProductRepo) InsertChangeRecords(tx *sql.Tx, changes []model.ChangeRecord) error {
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
