package storage

import "ppewh/internal/models"

type ItemsRepository interface {
	List() ([]models.Item, error)
	GetByID(id string) (models.Item, bool, error)
	Add(item models.Item) error
	DecreaseQuantityIfEnough(itemID string, qty int) (models.Item, error)
	IncreaseQuantity(itemID string, qty int) error
}

type UsersRepository interface {
	List() ([]models.User, error)
	GetByID(id string) (models.User, bool, error)
	Add(user models.User) error
}

type DepartmentsRepository interface {
	List() ([]models.Department, error)
	GetByID(id string) (models.Department, bool, error)
	Add(dept models.Department) error
}

type TransactionsRepository interface {
	List() ([]models.Transaction, error)
	GetByID(id string) (models.Transaction, bool, error)
	Add(tx models.Transaction) error
}

type ReturnsRepository interface {
	ListByTransactionID(transactionID string) ([]models.Return, error)
	List() ([]models.Return, error)
	Add(ret models.Return) error
}

type AuthAccountsRepository interface {
	List() ([]models.AuthAccount, error)
	GetByID(id string) (models.AuthAccount, bool, error)
	GetByEmail(email string) (models.AuthAccount, bool, error)
	Add(account models.AuthAccount) error
	Update(account models.AuthAccount) error
}

