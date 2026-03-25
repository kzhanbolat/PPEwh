package storage

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
	"sync"

	"ppewh/internal/models"
)

type CSVReturnsRepository struct {
	filePath string
	mu       *sync.Mutex
}

func newCSVReturnsRepository(filePath string, mu *sync.Mutex) *CSVReturnsRepository {
	return &CSVReturnsRepository{filePath: filePath, mu: mu}
}

func (r *CSVReturnsRepository) List() ([]models.Return, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readAll()
}

func (r *CSVReturnsRepository) ListByTransactionID(transactionID string) ([]models.Return, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	out := make([]models.Return, 0, len(all))
	for _, ret := range all {
		if ret.TransactionID == transactionID {
			out = append(out, ret)
		}
	}
	return out, nil
}

func (r *CSVReturnsRepository) Add(ret models.Return) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rets, err := r.readAll()
	if err != nil {
		return err
	}
	for _, t := range rets {
		if t.ID == ret.ID {
			return errors.New("return id already exists")
		}
	}
	rets = append(rets, ret)
	return r.writeAll(rets)
}

func (r *CSVReturnsRepository) readAll() ([]models.Return, error) {
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
		return []models.Return{}, nil
	}

	// Header: id,transaction_id,item_id,quantity_returned,returned_by_user_id,received_by_user_id,department_id,timestamp
	out := make([]models.Return, 0, len(records)-1)
	for idx := 1; idx < len(records); idx++ {
		rec := records[idx]
		if len(rec) < 8 {
			continue
		}

		qty, err := strconv.Atoi(rec[3])
		if err != nil {
			return nil, err
		}
		out = append(out, models.Return{
			ID:                rec[0],
			TransactionID:     rec[1],
			ItemID:            rec[2],
			QuantityReturned: qty,
			ReturnedByUserID: rec[4],
			ReceivedByUserID: rec[5],
			DepartmentID:      rec[6],
			Timestamp:         rec[7],
		})
	}
	return out, nil
}

func (r *CSVReturnsRepository) writeAll(returns []models.Return) error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{
		"id",
		"transaction_id",
		"item_id",
		"quantity_returned",
		"returned_by_user_id",
		"received_by_user_id",
		"department_id",
		"timestamp",
	}); err != nil {
		return err
	}

	for _, ret := range returns {
		if err := w.Write([]string{
			ret.ID,
			ret.TransactionID,
			ret.ItemID,
			strconv.Itoa(ret.QuantityReturned),
			ret.ReturnedByUserID,
			ret.ReceivedByUserID,
			ret.DepartmentID,
			ret.Timestamp,
		}); err != nil {
			return err
		}
	}
	return w.Error()
}

