package services

import (
	"errors"
	"strings"
	"time"

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
	if issueDate != "" && expiryDate != "" {
		issueAt, err := time.Parse("2006-01-02", issueDate)
		if err != nil {
			return models.Item{}, errors.New("issue date must be in YYYY-MM-DD format")
		}
		expiryAt, err := time.Parse("2006-01-02", expiryDate)
		if err != nil {
			return models.Item{}, errors.New("expiry date must be in YYYY-MM-DD format")
		}
		if !issueAt.Before(expiryAt) {
			return models.Item{}, errors.New("issue date must be earlier than expiry date")
		}
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

