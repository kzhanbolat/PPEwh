package models

// Return represents a PPE return against an existing issue transaction.
// Each row is linked to the original issue via TransactionID.
type Return struct {
	ID                string
	TransactionID    string
	ItemID            string
	QuantityReturned int
	ReturnedByUserID string // employee who returns PPE
	ReceivedByUserID string // warehouse staff who accepts return
	DepartmentID      string // derived from the original issue's issued_to user
	Timestamp         string // string format for simplicity
}

