package models

// AuthAccount represents platform credentials and access flags.
type AuthAccount struct {
	ID                string
	Name              string
	Email             string
	PasswordHash      string
	IsWarehouseWorker bool
	IsApproved        bool
	IsAdmin           bool
	MustResetPassword bool
	CreatedAt         string
	UpdatedAt         string
}
