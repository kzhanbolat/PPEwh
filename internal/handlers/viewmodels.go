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

