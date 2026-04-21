package storage

import (
	"encoding/csv"
	"errors"
	"os"
	"sync"

	"ppewh/internal/models"
)

type CSVUsersRepository struct {
	filePath string
	mu       *sync.Mutex
}

func newCSVUsersRepository(filePath string, mu *sync.Mutex) *CSVUsersRepository {
	return &CSVUsersRepository{filePath: filePath, mu: mu}
}

func (r *CSVUsersRepository) List() ([]models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.readAll()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *CSVUsersRepository) GetByID(id string) (models.User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.readAll()
	if err != nil {
		return models.User{}, false, err
	}
	for _, u := range users {
		if u.ID == id {
			return u, true, nil
		}
	}
	return models.User{}, false, nil
}

func (r *CSVUsersRepository) Add(user models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.readAll()
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.ID == user.ID {
			return errors.New("user id already exists")
		}
	}

	users = append(users, user)
	return r.writeAll(users)
}

func (r *CSVUsersRepository) readAll() ([]models.User, error) {
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
		return []models.User{}, nil
	}

	// Header: id,employee_id,name,department_id,role
	out := make([]models.User, 0, len(records)-1)
	for idx := 1; idx < len(records); idx++ {
		rec := records[idx]
		if len(rec) < 4 {
			continue
		}
		employeeID := ""
		name := rec[1]
		departmentID := rec[2]
		role := rec[3]
		// Backward compatibility with old csv format: id,name,department_id,role
		if len(rec) >= 5 {
			employeeID = rec[1]
			name = rec[2]
			departmentID = rec[3]
			role = rec[4]
		}
		out = append(out, models.User{
			ID:           rec[0],
			EmployeeID:   employeeID,
			Name:         name,
			DepartmentID: departmentID,
			Role:         role,
		})
	}
	return out, nil
}

func (r *CSVUsersRepository) writeAll(users []models.User) error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"id", "employee_id", "name", "department_id", "role"}); err != nil {
		return err
	}
	for _, u := range users {
		if err := w.Write([]string{u.ID, u.EmployeeID, u.Name, u.DepartmentID, u.Role}); err != nil {
			return err
		}
	}
	return w.Error()
}

