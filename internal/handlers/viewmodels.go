package handlers

import "ppewh/internal/models"

type ItemsTableData struct {
	Lang    string
	T       func(string) string
	Items    []models.Item
	Success  string
	Error    string
}

type UsersTableData struct {
	Lang    string
	T       func(string) string
	Users    []models.User
	DepartmentOptions []models.Department
	DepartmentNameByID map[string]string
	IsAdmin bool
	Success  string
	Error    string
}

type TransactionRow struct {
	EventType        string // "issue" or "return"
	Timestamp        string
	ItemName         string
	Quantity         int
	IssuedToUserID   string // issue.issued_to_user_id OR return.returned_by_user_id
	IssuedToUserName string
	IssuedByUserName string
	DepartmentName   string
}

type TransactionsTableData struct {
	Lang    string
	T       func(string) string
	Transactions []TransactionRow
	Flash        string
	FlashError   string
}

type ReturnIssueOption struct {
	TransactionID string
	Label         string
	RemainingQty  int
}

type IssueFormData struct {
	IssuedToUserID string
	IssuedToUserName string
	IssuedByUserID string
	ItemID         string
	Quantity       string
}

type ReturnFormData struct {
	TransactionID    string
	ReturnedByUserName string
	QuantityReturned string
	ReceivedByUserID string
}

type AuthLoginPageData struct {
	Lang    string
	T       func(string) string
	Email   string
	Success string
	Error   string
}

type AuthRegisterPageData struct {
	Lang              string
	T                 func(string) string
	Name              string
	Email             string
	IsWarehouseWorker bool
	Success           string
	Error             string
}

type AuthChangePasswordPageData struct {
	Lang    string
	T       func(string) string
	Success string
	Error   string
}

type AdminAccessPageData struct {
	Lang     string
	T        func(string) string
	Accounts []models.AuthAccount
	Success  string
	Error    string
}

