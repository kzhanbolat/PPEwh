package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ppewh/internal/models"
	"ppewh/internal/services"
)

type TransactionsHandler struct {
	itemsSvc *services.ItemsService
	txSvc    *services.TransactionsService
	usersSvc *services.UsersService
	deptsSvc *services.DepartmentsService
	// For MVP: request supplies issuer and receiver (no auth yet).
}

func NewTransactionsHandler(
	itemsSvc *services.ItemsService,
	txSvc *services.TransactionsService,
	usersSvc *services.UsersService,
	deptsSvc *services.DepartmentsService,
	_ any,
) *TransactionsHandler {
	return &TransactionsHandler{itemsSvc: itemsSvc, txSvc: txSvc, usersSvc: usersSvc, deptsSvc: deptsSvc}
}

func (h *TransactionsHandler) History(c *gin.Context) {
	_, rows, err := h.listTransactionRows()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "transactions.html", TransactionsTableData{
			Transactions: rows,
			FlashError:   "failed to load transactions",
		})
		return
	}

	// Server-side filtering (minimal).
	userName := strings.ToLower(c.Query("user_name"))
	issuedBy := strings.ToLower(c.Query("issued_by"))
	itemName := strings.ToLower(c.Query("item_name"))
	txType := c.Query("type")

	filtered := make([]TransactionRow, 0, len(rows))
	for _, t := range rows {
		if userName != "" && !strings.Contains(strings.ToLower(t.IssuedToUserName), userName) {
			continue
		}
		if issuedBy != "" && !strings.Contains(strings.ToLower(t.IssuedByUserName), issuedBy) {
			continue
		}
		if itemName != "" && !strings.Contains(strings.ToLower(t.ItemName), itemName) {
			continue
		}
		if txType != "" {
			// UI sends ISSUE/RETURN, model stores "issue"/"return".
			if strings.ToUpper(t.EventType) != txType {
				continue
			}
		}
		filtered = append(filtered, t)
	}

	c.HTML(http.StatusOK, "transactions.html", TransactionsTableData{
		Transactions: filtered,
	})
}

func (h *TransactionsHandler) Issue(c *gin.Context) {
	issueForm := IssueFormData{
		IssuedToUserID: c.PostForm("issued_to_user_id"),
		IssuedByUserID: c.PostForm("issued_by_user_id"),
		ItemID:         c.PostForm("item_id"),
		Quantity:       c.PostForm("quantity"),
	}
	if issueForm.Quantity == "" {
		issueForm.Quantity = "1"
	}

	qty, err := strconv.Atoi(issueForm.Quantity)
	if err != nil {
		h.renderDashboardRoot(c, http.StatusBadRequest, issueForm, ReturnFormData{QuantityReturned: "1"}, "quantity must be a number", "", "")
		return
	}

	_, err = h.txSvc.IssueItem(issueForm.ItemID, qty, issueForm.IssuedToUserID, issueForm.IssuedByUserID)
	if err != nil {
		// DO NOT redirect on error; render same page and preserve user input.
		h.renderDashboardRoot(c, http.StatusBadRequest, issueForm, ReturnFormData{QuantityReturned: "1"}, err.Error(), "", "")
		return
	}

	// Success: keep user on same page; reset issue form state.
	if c.GetHeader("HX-Request") != "" {
		empty := IssueFormData{Quantity: "1"}
		h.renderDashboardRoot(c, http.StatusOK, empty, ReturnFormData{QuantityReturned: "1"}, "", "Issue recorded successfully.", "")
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (h *TransactionsHandler) Return(c *gin.Context) {
	transactionID := c.PostForm("transaction_id")
	qtyStr := c.PostForm("quantity")
	receivedByUserID := c.PostForm("received_by_user_id")

	qty, err := strconv.Atoi(qtyStr)
	if err != nil {
		h.renderDashboardTransactions(c, "failed to parse quantity", http.StatusBadRequest)
		return
	}

	issueTx, ok, err := h.txSvc.GetByID(transactionID)
	if err != nil {
		h.renderDashboardTransactions(c, err.Error(), http.StatusBadRequest)
		return
	}
	if !ok {
		h.renderDashboardTransactions(c, "transaction not found", http.StatusBadRequest)
		return
	}

	// For MVP: returnedByUserID is the employee who originally received the PPE.
	returnedByUserID := issueTx.IssuedToUserID

	_, err = h.txSvc.ReturnItem(transactionID, qty, returnedByUserID, receivedByUserID)
	if err != nil {
		// For simplicity, re-render the dashboard root with an error (issue form reset).
		empty := IssueFormData{Quantity: "1"}
		returnForm := ReturnFormData{
			TransactionID:    transactionID,
			QuantityReturned: qtyStr,
			ReceivedByUserID: receivedByUserID,
		}
		h.renderDashboardRoot(c, http.StatusBadRequest, empty, returnForm, err.Error(), "", err.Error())
		return
	}

	if c.GetHeader("HX-Request") != "" {
		empty := IssueFormData{Quantity: "1"}
		returnForm := ReturnFormData{QuantityReturned: "1"}
		h.renderDashboardRoot(c, http.StatusOK, empty, returnForm, "", "Return recorded successfully.", "")
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (h *TransactionsHandler) renderDashboardTransactions(c *gin.Context, message string, status int) {
	_, rows, err := h.listTransactionRows()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard_transactions.html", TransactionsTableData{
			Transactions: []TransactionRow{},
			FlashError:   "failed to load transactions",
		})
		return
	}

	data := TransactionsTableData{Transactions: rows}
	if status >= 400 {
		data.FlashError = message
	} else {
		data.Flash = message
	}

	c.HTML(status, "dashboard_transactions.html", data)
}

func (h *TransactionsHandler) listTransactionRows() ([]models.Transaction, []TransactionRow, error) {
	// Load lookup data for human-readable table columns.
	users, err := h.usersSvc.List()
	if err != nil {
		return nil, nil, err
	}
	depts, err := h.deptsSvc.List()
	if err != nil {
		return nil, nil, err
	}
	txs, err := h.txSvc.List()
	if err != nil {
		return nil, nil, err
	}
	rets, err := h.txSvc.ListReturns()
	if err != nil {
		return nil, nil, err
	}

	userMap := map[string]models.User{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	deptMap := map[string]models.Department{}
	for _, d := range depts {
		deptMap[d.ID] = d
	}

	issueByID := map[string]models.Transaction{}
	for _, tx := range txs {
		issueByID[tx.ID] = tx
	}

	rows := make([]TransactionRow, 0, len(txs))
	for _, tx := range txs {
		toUser := userMap[tx.IssuedToUserID]
		byUser := userMap[tx.IssuedByUserID]
		dept := deptMap[tx.DepartmentID]
		rows = append(rows, TransactionRow{
			EventType:         "issue",
			Timestamp:        tx.Timestamp,
			ItemName:         tx.ItemName,
			Quantity:         tx.Quantity,
			IssuedToUserID:   tx.IssuedToUserID,
			IssuedToUserName: toUser.Name,
			IssuedByUserName: byUser.Name,
			DepartmentName:   dept.Name,
		})
	}

	for _, ret := range rets {
		toUser := userMap[ret.ReturnedByUserID]
		byUser := userMap[ret.ReceivedByUserID]
		dept := deptMap[ret.DepartmentID]

		issueTx := issueByID[ret.TransactionID]
		itemName := issueTx.ItemName

		rows = append(rows, TransactionRow{
			EventType:         "return",
			Timestamp:        ret.Timestamp,
			ItemName:         itemName,
			Quantity:         ret.QuantityReturned,
			IssuedToUserID:   ret.ReturnedByUserID,
			IssuedToUserName: toUser.Name,
			IssuedByUserName: byUser.Name,
			DepartmentName:   dept.Name,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Timestamp > rows[j].Timestamp
	})

	return txs, rows, nil
}

func (h *TransactionsHandler) renderDashboardRoot(c *gin.Context, status int, issueForm IssueFormData, returnForm ReturnFormData, errMsg string, flash string, flashError string) {
	data, renderErr := h.buildDashboardPageData(issueForm, returnForm, errMsg, flash, flashError)
	if renderErr != nil {
		c.String(http.StatusInternalServerError, "failed to render dashboard")
		return
	}

	page := PageData{
		Success: flash,
		Error:   errMsg,
		Data:    data,
	}
	if page.Error == "" && flashError != "" {
		page.Error = flashError
	}

	if c.GetHeader("HX-Request") != "" {
		// When HTMX swaps, return only the dashboard-root fragment (forms + transaction history).
		c.HTML(status, "dashboard_root.html", page)
		return
	}
	c.HTML(status, "index.html", page)
}

func (h *TransactionsHandler) buildDashboardPageData(issueForm IssueFormData, returnForm ReturnFormData, errMsg string, flash string, flashError string) (DashboardPageData, error) {
	items, err := h.itemsSvc.List()
	if err != nil {
		return DashboardPageData{}, err
	}
	users, err := h.usersSvc.List()
	if err != nil {
		return DashboardPageData{}, err
	}

	_, rows, err := h.listTransactionRows()
	if err != nil {
		return DashboardPageData{}, err
	}

	homeTmp := &HomeHandler{txSvc: h.txSvc, usersSvc: h.usersSvc}
	returnOptions := homeTmp.buildReturnIssueOptions(users, rows)

	pageData := DashboardPageData{
		Items:               items,
		IssuedToUsers:      filterUsers(users, false),
		WarehouseStaff:     filterUsers(users, true),
		ReturnIssueOptions: returnOptions,
		Error:               errMsg,
		IssueForm:           issueForm,
		ReturnForm:         returnForm,
		TransactionsTableData: TransactionsTableData{
			Transactions: rows,
			Flash:        flash,
			FlashError:   flashError,
		},
	}

	if pageData.IssueForm.Quantity == "" {
		pageData.IssueForm.Quantity = "1"
	}

	return pageData, nil
}

