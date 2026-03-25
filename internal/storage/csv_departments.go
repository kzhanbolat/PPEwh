package storage

import (
	"encoding/csv"
	"errors"
	"os"
	"sync"

	"ppewh/internal/models"
)

type CSVDepartmentsRepository struct {
	filePath string
	mu       *sync.Mutex
}

func newCSVDepartmentsRepository(filePath string, mu *sync.Mutex) *CSVDepartmentsRepository {
	return &CSVDepartmentsRepository{filePath: filePath, mu: mu}
}

func (r *CSVDepartmentsRepository) List() ([]models.Department, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.readAll()
}

func (r *CSVDepartmentsRepository) GetByID(id string) (models.Department, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	depts, err := r.readAll()
	if err != nil {
		return models.Department{}, false, err
	}
	for _, d := range depts {
		if d.ID == id {
			return d, true, nil
		}
	}
	return models.Department{}, false, nil
}

func (r *CSVDepartmentsRepository) Add(dept models.Department) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	depts, err := r.readAll()
	if err != nil {
		return err
	}
	for _, d := range depts {
		if d.ID == dept.ID {
			return errors.New("department id already exists")
		}
	}

	depts = append(depts, dept)
	return r.writeAll(depts)
}

func (r *CSVDepartmentsRepository) readAll() ([]models.Department, error) {
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
		return []models.Department{}, nil
	}

	// Header: id,name
	out := make([]models.Department, 0, len(records)-1)
	for idx := 1; idx < len(records); idx++ {
		rec := records[idx]
		if len(rec) < 2 {
			continue
		}
		out = append(out, models.Department{
			ID:   rec[0],
			Name: rec[1],
		})
	}
	return out, nil
}

func (r *CSVDepartmentsRepository) writeAll(depts []models.Department) error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"id", "name"}); err != nil {
		return err
	}
	for _, d := range depts {
		if err := w.Write([]string{d.ID, d.Name}); err != nil {
			return err
		}
	}
	return w.Error()
}

