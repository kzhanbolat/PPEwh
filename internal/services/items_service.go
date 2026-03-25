package services

import (
	"errors"
	"strings"

	"ppewh/internal/models"
	"ppewh/internal/storage"
)

type ItemsService struct {
	repo storage.ItemsRepository
}

func NewItemsService(repo storage.ItemsRepository) *ItemsService {
	return &ItemsService{repo: repo}
}

func (s *ItemsService) List() ([]models.Item, error) {
	return s.repo.List()
}

func (s *ItemsService) AddItem(name, size string, quantity int, issueDate, expiryDate string) (models.Item, error) {
	name = strings.TrimSpace(name)
	size = strings.TrimSpace(size)
	issueDate = strings.TrimSpace(issueDate)
	expiryDate = strings.TrimSpace(expiryDate)
	if name == "" {
		return models.Item{}, errors.New("item name is required")
	}
	if size == "" {
		return models.Item{}, errors.New("item size is required")
	}
	if quantity <= 0 {
		return models.Item{}, errors.New("quantity must be > 0")
	}

	item := models.Item{
		ID:       NewID("ITEM"),
		Name:     name,
		Size:     size,
		Quantity: quantity,
		IssueDate:  issueDate,
		ExpiryDate: expiryDate,
	}
	if err := s.repo.Add(item); err != nil {
		return models.Item{}, err
	}
	return item, nil
}

