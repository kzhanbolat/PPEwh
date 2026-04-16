package services

import (
	"errors"
	"strings"
	"time"

	"ppewh/internal/models"
	"ppewh/internal/storage"
)

type AuthService struct {
	repo   storage.AuthAccountsRepository
	hasher *PasswordHasher
}

func NewAuthService(repo storage.AuthAccountsRepository, hasher *PasswordHasher) *AuthService {
	return &AuthService{repo: repo, hasher: hasher}
}

func (s *AuthService) Register(name, email, password string, isWarehouseWorker bool) error {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if name == "" {
		return errors.New("name is required")
	}
	if email == "" {
		return errors.New("email is required")
	}
	if len(password) < 4 {
		return errors.New("password must be at least 4 characters")
	}
	if !isWarehouseWorker {
		return errors.New("only warehouse employees can register")
	}

	if _, exists, err := s.repo.GetByEmail(email); err != nil {
		return err
	} else if exists {
		return errors.New("email already registered")
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return errors.New("failed to hash password")
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	account := models.AuthAccount{
		ID:                NewID("ACC"),
		Name:              name,
		Email:             email,
		PasswordHash:      string(hash),
		IsWarehouseWorker: true,
		IsApproved:        false,
		IsAdmin:           false,
		MustResetPassword: false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return s.repo.Add(account)
}

func (s *AuthService) Authenticate(email, password string) (models.AuthAccount, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if email == "" || password == "" {
		return models.AuthAccount{}, errors.New("email and password are required")
	}

	account, ok, err := s.repo.GetByEmail(email)
	if err != nil {
		return models.AuthAccount{}, err
	}
	if !ok {
		return models.AuthAccount{}, errors.New("invalid credentials")
	}
	if !s.hasher.Verify(account.PasswordHash, password) {
		return models.AuthAccount{}, errors.New("invalid credentials")
	}
	if !account.IsAdmin && !account.IsApproved {
		return models.AuthAccount{}, errors.New("access is pending admin approval")
	}
	return account, nil
}

func (s *AuthService) ListAccounts() ([]models.AuthAccount, error) {
	return s.repo.List()
}

func (s *AuthService) GetByID(id string) (models.AuthAccount, bool, error) {
	return s.repo.GetByID(id)
}

func (s *AuthService) SetApproval(accountID string, approved bool) error {
	account, ok, err := s.repo.GetByID(accountID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("account not found")
	}
	account.IsApproved = approved
	account.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(account)
}

func (s *AuthService) ResetPasswordByAdmin(accountID, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 4 {
		return errors.New("new password must be at least 4 characters")
	}

	account, ok, err := s.repo.GetByID(accountID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("account not found")
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return errors.New("failed to hash password")
	}
	account.PasswordHash = string(hash)
	account.MustResetPassword = true
	account.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(account)
}

func (s *AuthService) ChangePassword(accountID, currentPassword, newPassword string) error {
	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 4 {
		return errors.New("new password must be at least 4 characters")
	}

	account, ok, err := s.repo.GetByID(accountID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("account not found")
	}
	if !s.hasher.Verify(account.PasswordHash, currentPassword) {
		return errors.New("current password is incorrect")
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return errors.New("failed to hash password")
	}
	account.PasswordHash = string(hash)
	account.MustResetPassword = false
	account.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Update(account)
}

func (s *AuthService) EnsureDefaultAdmin() error {
	if _, exists, err := s.repo.GetByEmail("admin@ppe.local"); err != nil {
		return err
	} else if exists {
		return nil
	}

	hash, err := s.hasher.Hash("admin")
	if err != nil {
		return err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return s.repo.Add(models.AuthAccount{
		ID:                "ACC-ADMIN",
		Name:              "Administrator",
		Email:             "admin@ppe.local",
		PasswordHash:      hash,
		IsWarehouseWorker: true,
		IsApproved:        true,
		IsAdmin:           true,
		MustResetPassword: false,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
}
