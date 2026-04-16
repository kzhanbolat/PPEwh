package storage

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"

	"ppewh/internal/models"
)

type CSVAuthAccountsRepository struct {
	filePath string
	mu       *sync.Mutex
}

func newCSVAuthAccountsRepository(filePath string, mu *sync.Mutex) *CSVAuthAccountsRepository {
	return &CSVAuthAccountsRepository{filePath: filePath, mu: mu}
}

func (r *CSVAuthAccountsRepository) List() ([]models.AuthAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readAll()
}

func (r *CSVAuthAccountsRepository) GetByID(id string) (models.AuthAccount, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	accounts, err := r.readAll()
	if err != nil {
		return models.AuthAccount{}, false, err
	}
	for _, a := range accounts {
		if a.ID == id {
			return a, true, nil
		}
	}
	return models.AuthAccount{}, false, nil
}

func (r *CSVAuthAccountsRepository) GetByEmail(email string) (models.AuthAccount, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	needle := strings.ToLower(strings.TrimSpace(email))
	accounts, err := r.readAll()
	if err != nil {
		return models.AuthAccount{}, false, err
	}
	for _, a := range accounts {
		if strings.ToLower(strings.TrimSpace(a.Email)) == needle {
			return a, true, nil
		}
	}
	return models.AuthAccount{}, false, nil
}

func (r *CSVAuthAccountsRepository) Add(account models.AuthAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accounts, err := r.readAll()
	if err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(account.Email))
	for _, a := range accounts {
		if a.ID == account.ID {
			return errors.New("account id already exists")
		}
		if strings.ToLower(strings.TrimSpace(a.Email)) == email {
			return errors.New("email already exists")
		}
	}

	accounts = append(accounts, account)
	return r.writeAll(accounts)
}

func (r *CSVAuthAccountsRepository) Update(account models.AuthAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accounts, err := r.readAll()
	if err != nil {
		return err
	}

	email := strings.ToLower(strings.TrimSpace(account.Email))
	found := false
	for i, a := range accounts {
		if a.ID == account.ID {
			accounts[i] = account
			found = true
			continue
		}
		if strings.ToLower(strings.TrimSpace(a.Email)) == email {
			return errors.New("email already exists")
		}
	}
	if !found {
		return errors.New("account not found")
	}
	return r.writeAll(accounts)
}

func (r *CSVAuthAccountsRepository) readAll() ([]models.AuthAccount, error) {
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
		return []models.AuthAccount{}, nil
	}

	out := make([]models.AuthAccount, 0, len(records)-1)
	for idx := 1; idx < len(records); idx++ {
		rec := records[idx]
		if len(rec) < 10 {
			continue
		}
		isWarehouse, _ := strconv.ParseBool(rec[4])
		isApproved, _ := strconv.ParseBool(rec[5])
		isAdmin, _ := strconv.ParseBool(rec[6])
		mustReset, _ := strconv.ParseBool(rec[7])
		out = append(out, models.AuthAccount{
			ID:                rec[0],
			Name:              rec[1],
			Email:             rec[2],
			PasswordHash:      rec[3],
			IsWarehouseWorker: isWarehouse,
			IsApproved:        isApproved,
			IsAdmin:           isAdmin,
			MustResetPassword: mustReset,
			CreatedAt:         rec[8],
			UpdatedAt:         rec[9],
		})
	}
	return out, nil
}

func (r *CSVAuthAccountsRepository) writeAll(accounts []models.AuthAccount) error {
	f, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{
		"id", "name", "email", "password_hash", "is_warehouse_worker", "is_approved", "is_admin", "must_reset_password", "created_at", "updated_at",
	}); err != nil {
		return err
	}
	for _, a := range accounts {
		if err := w.Write([]string{
			a.ID,
			a.Name,
			strings.ToLower(strings.TrimSpace(a.Email)),
			a.PasswordHash,
			strconv.FormatBool(a.IsWarehouseWorker),
			strconv.FormatBool(a.IsApproved),
			strconv.FormatBool(a.IsAdmin),
			strconv.FormatBool(a.MustResetPassword),
			a.CreatedAt,
			a.UpdatedAt,
		}); err != nil {
			return err
		}
	}
	return w.Error()
}
