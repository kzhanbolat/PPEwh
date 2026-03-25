package handlers

import "ppewh/internal/models"

type ItemsTableData struct {
	Items    []models.Item
	Success  string
	Error    string
}

type UsersTableData struct {
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
	IssuedByUserID string
	ItemID         string
	Quantity       string
}

type ReturnFormData struct {
	TransactionID    string
	QuantityReturned string
	ReceivedByUserID string
}

