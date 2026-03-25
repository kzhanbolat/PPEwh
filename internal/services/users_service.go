package services

import (
	"errors"
	"strings"

	"ppewh/internal/models"
	"ppewh/internal/storage"
)

type UsersService struct {
	repo storage.UsersRepository
}

func NewUsersService(repo storage.UsersRepository) *UsersService {
	return &UsersService{repo: repo}
}

func (s *UsersService) List() ([]models.User, error) {
	return s.repo.List()
}

func (s *UsersService) AddUser(name, departmentID, role string) (models.User, error) {
	name = strings.TrimSpace(name)
	departmentID = strings.TrimSpace(departmentID)
	role = strings.TrimSpace(strings.ToLower(role))

	if name == "" {
		return models.User{}, errors.New("user name is required")
	}
	if departmentID == "" {
		return models.User{}, errors.New("department_id is required")
	}
	switch role {
	case "employee", "warehouse":
		// ok
	default:
		return models.User{}, errors.New(`role must be "employee" or "warehouse"`)
	}

	user := models.User{
		ID:           NewID("USR"),
		Name:         name,
		DepartmentID: departmentID,
		Role:         role,
	}
	if err := s.repo.Add(user); err != nil {
		return models.User{}, err
	}
	return user, nil
}

