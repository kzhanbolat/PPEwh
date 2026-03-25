package storage

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
	"sync"

	"ppewh/internal/models"
)

type CSVTransactionsRepository struct {
	filePath string
	mu       *sync.Mutex
}

func newCSVTransactionsRepository(filePath string, mu *sync.Mutex) *CSVTransactionsRepository {
	return &CSVTransactionsRepository{filePath: filePath, mu: mu}
}

func (r *CSVTransactionsRepository) List() ([]models.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.readAll()
}

func (r *CSVTransactionsRepository) GetByID(id string) (models.Transaction, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	txs, err := r.readAll()
	if err != nil {
		return models.Transaction{}, false, err
	}
	for _, tx := range txs {
		if tx.ID == id {
			return tx, true, nil
		}
	}
	return models.Transaction{}, false, nil
}

func (r *CSVTransactionsRepository) Add(tx models.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	transactions, err := r.readAll()
	if err != nil {
		return err
	}

	for _, t := range transactions {
		if t.ID == tx.ID {
			return errors.New("transaction id already exists")
		}
	}

	transactions = append(transactions, tx)
	return r.writeAll(transactions)
}

func (r *CSVTransactionsRepository) readAll() ([]models.Transaction, error) {
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
		return []models.Transaction{}, nil
	}

	// Header: id,item_id,item_name,quantity,issued_to_user_id,issued_by_user_id,department_id,timestamp
	out := make([]models.Transaction, 0, len(records)-1)
	for idx := 1; idx < len(records); idx++ {
		rec := records[idx]
		if len(rec) < 8 {
			continue
		}

		qty, err := strconv.Atoi(rec[3])
		if err != nil {
			return nil, err
		}
		out = append(out, models.Transaction{
			ID:             rec[0],
			ItemID:         rec[1],
			ItemName:       rec[2],
			Quantity:       qty,
			IssuedToUserID: rec[4],
			IssuedByUserID: rec[5],
			DepartmentID:   rec[6],
			Timestamp:      rec[7],
		})
	}
	return out, nil
}

func (r *CSVTransactionsRepository) writeAll(transactions []models.Transaction) error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"id", "item_id", "item_name", "quantity", "issued_to_user_id", "issued_by_user_id", "department_id", "timestamp"}); err != nil {
		return err
	}
	for _, t := range transactions {
		if err := w.Write([]string{
			t.ID,
			t.ItemID,
			t.ItemName,
			strconv.Itoa(t.Quantity),
			t.IssuedToUserID,
			t.IssuedByUserID,
			t.DepartmentID,
			t.Timestamp,
		}); err != nil {
			return err
		}
	}

	return w.Error()
}

