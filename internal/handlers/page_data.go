package handlers

// PageData is a unified container for page-level success/error messages.
// Data holds the concrete view model (e.g. DashboardPageData, TransactionsTableData, ...).
type PageData struct {
	Lang    string
	T       func(string) string
	Success string
	Error   string
	Data    any
}

