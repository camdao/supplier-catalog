package service

import (
	"catalog/internal/model"
	"catalog/internal/repo"
)

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
