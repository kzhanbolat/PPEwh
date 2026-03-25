package models

// Item represents a PPE inventory record.
type Item struct {
	ID       string
	Name     string
	Size     string
	Quantity int
	IssueDate  string // YYYY-MM-DD
	ExpiryDate string // YYYY-MM-DD
}

