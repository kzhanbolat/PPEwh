package storage

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
	"sync"

	"ppewh/internal/models"
)

type CSVItemsRepository struct {
	filePath string
	mu       *sync.Mutex
}

func newCSVItemsRepository(filePath string, mu *sync.Mutex) *CSVItemsRepository {
	return &CSVItemsRepository{filePath: filePath, mu: mu}
}

func (r *CSVItemsRepository) List() ([]models.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.readAll()
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CSVItemsRepository) GetByID(id string) (models.Item, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.readAll()
	if err != nil {
		return models.Item{}, false, err
	}
	for _, it := range items {
		if it.ID == id {
			return it, true, nil
		}
	}
	return models.Item{}, false, nil
}

func (r *CSVItemsRepository) Add(item models.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.readAll()
	if err != nil {
		return err
	}

	// Basic uniqueness guard (avoid duplicated IDs).
	for _, it := range items {
		if it.ID == item.ID {
			return errors.New("item id already exists")
		}
	}

	items = append(items, item)
	return r.writeAll(items)
}

// DecreaseQuantityIfEnough decreases quantity and prevents negative stock.
func (r *CSVItemsRepository) DecreaseQuantityIfEnough(itemID string, qty int) (models.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.readAll()
	if err != nil {
		return models.Item{}, err
	}

	for i := range items {
		if items[i].ID != itemID {
			continue
		}
		if qty < 0 {
			return models.Item{}, errors.New("qty must be non-negative")
		}
		if items[i].Quantity-qty < 0 {
			return models.Item{}, errors.New("insufficient stock")
		}
		items[i].Quantity -= qty
		if err := r.writeAll(items); err != nil {
			return models.Item{}, err
		}
		return items[i], nil
	}

	return models.Item{}, errors.New("item not found")
}

func (r *CSVItemsRepository) IncreaseQuantity(itemID string, qty int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if qty < 0 {
		return errors.New("qty must be non-negative")
	}

	items, err := r.readAll()
	if err != nil {
		return err
	}

	for i := range items {
		if items[i].ID != itemID {
			continue
		}
		items[i].Quantity += qty
		return r.writeAll(items)
	}

	return errors.New("item not found")
}

func (r *CSVItemsRepository) readAll() ([]models.Item, error) {
	f, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cr := csv.NewReader(f)
	cr.ReuseRecord = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return []models.Item{}, nil
	}

	// Header: id,name,size,quantity,issue_date,expiry_date
	out := make([]models.Item, 0, len(records)-1)
	for idx := 1; idx < len(records); idx++ {
		rec := records[idx]
		// tolerate malformed/short lines
		if len(rec) < 4 {
			continue
		}

		qty, err := strconv.Atoi(rec[3])
		if err != nil {
			return nil, err
		}

		issueDate := ""
		expiryDate := ""
		if len(rec) >= 6 {
			issueDate = rec[4]
			expiryDate = rec[5]
		}

		out = append(out, models.Item{
			ID:       rec[0],
			Name:     rec[1],
			Size:     rec[2],
			Quantity: qty,
			IssueDate:  issueDate,
			ExpiryDate: expiryDate,
		})
	}
	return out, nil
}

func (r *CSVItemsRepository) writeAll(items []models.Item) error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header: id,name,size,quantity,issue_date,expiry_date
	if err := w.Write([]string{"id", "name", "size", "quantity", "issue_date", "expiry_date"}); err != nil {
		return err
	}
	for _, it := range items {
		if err := w.Write([]string{it.ID, it.Name, it.Size, strconv.Itoa(it.Quantity), it.IssueDate, it.ExpiryDate}); err != nil {
			return err
		}
	}
	return w.Error()
}

