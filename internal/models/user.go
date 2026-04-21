package models

// User represents an employee that can RECEIVE PPE.
type User struct {
	ID           string
	EmployeeID   string
	Name         string
	DepartmentID string
	// Role controls who is allowed to ISSUE items (e.g. warehouse staff).
	// Common values: "employee" | "warehouse".
	Role string
}

