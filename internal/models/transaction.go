package models

// Transaction represents a PPE issuing record.
type Transaction struct {
	ID              string
	ItemID          string
	ItemName        string // Snapshot for CSV simplicity
	Quantity        int
	IssuedToUserID  string
	IssuedByUserID  string
	DepartmentID    string // derived from IssuedToUser
	Timestamp       string // string format, e.g. ISO-8601 or any display-friendly format
}

