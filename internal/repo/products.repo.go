package repo

import (
	"catalog/global"
	"catalog/internal/model"
	"database/sql"
)

type ProductsRepo struct{}

func NewProductsRepo() *ProductsRepo {
	return &ProductsRepo{}
}

func (r *ProductsRepo) ListProducts(category string, includeWithdrawn bool) ([]model.Product, error) {
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

func (r *ProductsRepo) GetProduct(id string) (*model.Product, error) {
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
