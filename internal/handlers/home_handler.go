package handlers

import (
	"net/http"
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"ppewh/internal/models"
	"ppewh/internal/services"
)

type DashboardPageData struct {
	Items []models.Item
	IssuedToUsers []models.User
	WarehouseStaff []models.User
	ReturnIssueOptions []ReturnIssueOption
	Error string
	IssueForm IssueFormData
	ReturnForm ReturnFormData
	TransactionsTableData
}

type HomeHandler struct {
	itemsSvc *services.ItemsService
	usersSvc *services.UsersService
	txSvc    *services.TransactionsService
	deptsSvc *services.DepartmentsService
}

func NewHomeHandler(itemsSvc *services.ItemsService, usersSvc *services.UsersService, txSvc *services.TransactionsService, deptsSvc *services.DepartmentsService) *HomeHandler {
	return &HomeHandler{itemsSvc: itemsSvc, usersSvc: usersSvc, txSvc: txSvc, deptsSvc: deptsSvc}
}

func (h *HomeHandler) Dashboard(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	c.HTML(http.StatusOK, "main.html", PageData{
		Lang:    lang,
		T:       t,
		Success: "",
		Error:   "",
		Data:    nil,
	})
}

func filterUsers(users []models.User, warehouseOnly bool) []models.User {
	out := make([]models.User, 0, len(users))
	for _, u := range users {
		isWarehouse := strings.EqualFold(strings.TrimSpace(u.Role), "warehouse")
		if warehouseOnly {
			if isWarehouse {
				out = append(out, u)
			}
		} else {
			// all employees receive PPE, including warehouse staff
			out = append(out, u)
		}
	}
	return out
}

func listTransactionsTableRows(txSvc *services.TransactionsService, usersSvc *services.UsersService, deptsSvc *services.DepartmentsService) TransactionsTableData {
	users, err := usersSvc.List()
	if err != nil {
		return TransactionsTableData{FlashError: "failed to load users for transactions"}
	}
	depts, err := deptsSvc.List()
	if err != nil {
		return TransactionsTableData{FlashError: "failed to load departments for transactions"}
	}
	txs, err := txSvc.List()
	if err != nil {
		return TransactionsTableData{FlashError: "failed to load transactions"}
	}
	rets, err := txSvc.ListReturns()
	if err != nil {
		return TransactionsTableData{FlashError: "failed to load returns"}
	}

	userMap := map[string]models.User{}
	for _, u := range users {
		userMap[u.ID] = u
	}
	deptMap := map[string]models.Department{}
	for _, d := range depts {
		deptMap[d.ID] = d
	}

	rows := make([]TransactionRow, 0, len(txs))
	issueByID := map[string]models.Transaction{}
	for _, tx := range txs {
		issueByID[tx.ID] = tx
	}
	for _, tx := range txs {
		toUser := userMap[tx.IssuedToUserID]
		byUser := userMap[tx.IssuedByUserID]
		dept := deptMap[tx.DepartmentID]
		toName := toUser.Name
		if toName == "" {
			toName = tx.IssuedToUserID
		}
		byName := byUser.Name
		if byName == "" {
			byName = tx.IssuedByUserID
		}
		rows = append(rows, TransactionRow{
			EventType:         "issue",
			Timestamp:        tx.Timestamp,
			ItemName:         tx.ItemName,
			Quantity:         tx.Quantity,
			IssuedToUserName: toName,
			IssuedByUserName: byName,
			DepartmentName:   dept.Name,
		})
	}

	for _, ret := range rets {
		toUser := userMap[ret.ReturnedByUserID]
		byUser := userMap[ret.ReceivedByUserID]
		dept := deptMap[ret.DepartmentID]
		issueTx := issueByID[ret.TransactionID]
		toName := toUser.Name
		if toName == "" {
			toName = ret.ReturnedByUserID
		}
		byName := byUser.Name
		if byName == "" {
			byName = ret.ReceivedByUserID
		}
		rows = append(rows, TransactionRow{
			EventType:         "return",
			Timestamp:        ret.Timestamp,
			ItemName:         issueTx.ItemName,
			Quantity:         ret.QuantityReturned,
			IssuedToUserName: toName,
			IssuedByUserName: byName,
			DepartmentName:   dept.Name,
		})
	}

	// ISO-8601 timestamps sort lexicographically.
	sort.Slice(rows, func(i, j int) bool { return rows[i].Timestamp > rows[j].Timestamp })

	return TransactionsTableData{Transactions: rows}
}

func (h *HomeHandler) buildReturnIssueOptions(allUsers []models.User, historyRows []TransactionRow) []ReturnIssueOption {
	// For the return form, we only offer issue transactions that still have remaining quantity.
	// We compute remaining using returns.csv, not the UI rows.
	issues, err := h.txSvc.List()
	if err != nil {
		return []ReturnIssueOption{}
	}
	rets, err := h.txSvc.ListReturns()
	if err != nil {
		return []ReturnIssueOption{}
	}

	userMap := map[string]models.User{}
	for _, u := range allUsers {
		userMap[u.ID] = u
	}

	returnedByTx := map[string]int{}
	for _, r := range rets {
		returnedByTx[r.TransactionID] += r.QuantityReturned
	}

	opts := make([]ReturnIssueOption, 0, len(issues))
	for _, tx := range issues {
		returnedQty := returnedByTx[tx.ID]
		remaining := tx.Quantity - returnedQty
		if remaining <= 0 {
			continue
		}
		toUser := userMap[tx.IssuedToUserID]
		label := fmt.Sprintf("%s - %s (Issued to: %s, Remaining: %d)", tx.ID, tx.ItemName, toUser.Name, remaining)
		opts = append(opts, ReturnIssueOption{
			TransactionID: tx.ID,
			Label:         label,
			RemainingQty:  remaining,
		})
	}

	// Newest first.
	sort.Slice(opts, func(i, j int) bool {
		return opts[i].TransactionID > opts[j].TransactionID
	})

	return opts
}

