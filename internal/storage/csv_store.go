package storage

import (
	"os"
	"path/filepath"
	"sync"

	"ppewh/internal/models"
)

// CSVStore wires repositories backed by CSV files.
type CSVStore struct {
	mu sync.Mutex

	itemsRepo        *CSVItemsRepository
	usersRepo        *CSVUsersRepository
	departmentsRepo  *CSVDepartmentsRepository
	transactionsRepo *CSVTransactionsRepository
	returnsRepo      *CSVReturnsRepository
}

func NewCSVStore(dataDir string) (*CSVStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	itemsPath := filepath.Join(dataDir, "items.csv")
	usersPath := filepath.Join(dataDir, "users.csv")
	departmentsPath := filepath.Join(dataDir, "departments.csv")
	transactionsPath := filepath.Join(dataDir, "transactions.csv")
	returnsPath := filepath.Join(dataDir, "returns.csv")

	s := &CSVStore{}
	s.itemsRepo = newCSVItemsRepository(itemsPath, &s.mu)
	s.usersRepo = newCSVUsersRepository(usersPath, &s.mu)
	s.departmentsRepo = newCSVDepartmentsRepository(departmentsPath, &s.mu)
	s.transactionsRepo = newCSVTransactionsRepository(transactionsPath, &s.mu)
	s.returnsRepo = newCSVReturnsRepository(returnsPath, &s.mu)

	// Preload headers + sample data if files don't exist.
	if err := s.ensureFiles(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *CSVStore) ensureFiles() error {
	// Items
	if _, err := os.Stat(s.itemsRepo.filePath); os.IsNotExist(err) {
		sample := []models.Item{
			{ID: "ITEM-001", Name: "Safety Helmet", Size: "One Size", Quantity: 25, IssueDate: "2026-03-01", ExpiryDate: "2027-03-01"},
			{ID: "ITEM-002", Name: "Safety Gloves", Size: "L", Quantity: 50, IssueDate: "2026-03-01", ExpiryDate: "2027-03-01"},
			{ID: "ITEM-003", Name: "Hi-Vis Vest", Size: "M", Quantity: 30, IssueDate: "2026-03-01", ExpiryDate: "2027-03-01"},
		}
		if err := s.itemsRepo.writeAll(sample); err != nil {
			return err
		}
	}

	// Users
	if _, err := os.Stat(s.usersRepo.filePath); os.IsNotExist(err) {
		sample := []models.User{
			{ID: "USR-001", Name: "Alice", DepartmentID: "DEPT-001", Role: "warehouse"},
			{ID: "USR-002", Name: "Bob", DepartmentID: "DEPT-002", Role: "employee"},
			{ID: "USR-003", Name: "Charlie", DepartmentID: "DEPT-003", Role: "employee"},
		}
		if err := s.usersRepo.writeAll(sample); err != nil {
			return err
		}
	}

	// Departments
	if _, err := os.Stat(s.departmentsRepo.filePath); os.IsNotExist(err) {
		sample := []models.Department{
			{ID: "DEPT-001", Name: "Warehouse"},
			{ID: "DEPT-002", Name: "Maintenance"},
			{ID: "DEPT-003", Name: "Logistics"},
		}
		if err := s.departmentsRepo.writeAll(sample); err != nil {
			return err
		}
	}

	// Transactions
	if _, err := os.Stat(s.transactionsRepo.filePath); os.IsNotExist(err) {
		// NOTE: issued_by_user_id is restricted to warehouse staff in business logic.
		sample := []models.Transaction{
			{
				ID:             "TX-001",
				ItemID:         "ITEM-001",
				ItemName:       "Safety Helmet",
				Quantity:       1,
				IssuedToUserID: "USR-002",
				IssuedByUserID: "USR-001",
				DepartmentID:   "DEPT-002",
				Timestamp:      "2026-03-10 09:00:00",
			},
			{
				ID:             "TX-002",
				ItemID:         "ITEM-002",
				ItemName:       "Safety Gloves",
				Quantity:       2,
				IssuedToUserID: "USR-003",
				IssuedByUserID: "USR-001",
				DepartmentID:   "DEPT-003",
				Timestamp:      "2026-03-12 10:30:00",
			},
		}
		if err := s.transactionsRepo.writeAll(sample); err != nil {
			return err
		}
	}

	// Returns
	if _, err := os.Stat(s.returnsRepo.filePath); os.IsNotExist(err) {
		// MVP default: no returns recorded yet.
		if err := s.returnsRepo.writeAll([]models.Return{}); err != nil {
			return err
		}
	}

	return nil
}

func (s *CSVStore) Items() *CSVItemsRepository { return s.itemsRepo }
func (s *CSVStore) Users() *CSVUsersRepository { return s.usersRepo }
func (s *CSVStore) Departments() *CSVDepartmentsRepository { return s.departmentsRepo }
func (s *CSVStore) Transactions() *CSVTransactionsRepository {
	return s.transactionsRepo
}

func (s *CSVStore) Returns() *CSVReturnsRepository {
	return s.returnsRepo
}

