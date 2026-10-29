package service

import (
	"catalog/internal/model"
	"catalog/internal/repo"
)

type ProductsService struct {
	Repo *repo.ProductsRepo
}

func NewProductsService() *ProductsService {
	return &ProductsService{
		Repo: repo.NewProductsRepo(),
	}
}

func (s *ProductsService) ListProducts(category string, includeWithdrawn bool) ([]model.Product, error) {
	return s.Repo.ListProducts(category, includeWithdrawn)
}

func (s *ProductsService) GetProduct(id string) (*model.Product, error) {
	return s.Repo.GetProduct(id)
}
