package services

import (
	"errors"
	"strings"

	"ppewh/internal/models"
	"ppewh/internal/storage"
)

type DepartmentsService struct {
	repo storage.DepartmentsRepository
}

func NewDepartmentsService(repo storage.DepartmentsRepository) *DepartmentsService {
	return &DepartmentsService{repo: repo}
}

func (s *DepartmentsService) List() ([]models.Department, error) {
	return s.repo.List()
}

func (s *DepartmentsService) AddDepartment(name string) (models.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Department{}, errors.New("department name is required")
	}
	dept := models.Department{
		ID:   NewID("DEPT"),
		Name: name,
	}
	if err := s.repo.Add(dept); err != nil {
		return models.Department{}, err
	}
	return dept, nil
}

