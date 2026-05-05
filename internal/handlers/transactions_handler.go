package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"ppewh/internal/models"
	"ppewh/internal/i18n"
	"ppewh/internal/services"
)

type TransactionsHandler struct {
	itemsSvc *services.ItemsService
	txSvc    *services.TransactionsService
	usersSvc *services.UsersService
	deptsSvc *services.DepartmentsService
	authSvc  *services.AuthService
}

func NewTransactionsHandler(
	itemsSvc *services.ItemsService,
	txSvc *services.TransactionsService,
	usersSvc *services.UsersService,
	deptsSvc *services.DepartmentsService,
	authSvc *services.AuthService,
	_ any,
) *TransactionsHandler {
	return &TransactionsHandler{itemsSvc: itemsSvc, txSvc: txSvc, usersSvc: usersSvc, deptsSvc: deptsSvc, authSvc: authSvc}
}

func (h *TransactionsHandler) History(c *gin.Context) {
	lang := getLang(c)
	tfunc := translator(lang)

	_, rows, err := h.listTransactionRows()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "transactions.html", TransactionsTableData{
			Lang:         lang,
			T:            tfunc,
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
		Lang:         lang,
		T:            tfunc,
		Transactions: filtered,
	})
}

func (h *TransactionsHandler) Export(c *gin.Context) {
	_, rows, err := h.listTransactionRows()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load transactions")
		return
	}

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
		if txType != "" && strings.ToUpper(t.EventType) != txType {
			continue
		}
		filtered = append(filtered, t)
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	headers := []string{"timestamp", "type", "issued_to", "issued_by", "department", "item", "quantity"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, row := range filtered {
		r := i + 2
		_ = f.SetCellValue(sheet, "A"+strconv.Itoa(r), row.Timestamp)
		_ = f.SetCellValue(sheet, "B"+strconv.Itoa(r), row.EventType)
		_ = f.SetCellValue(sheet, "C"+strconv.Itoa(r), row.IssuedToUserName)
		_ = f.SetCellValue(sheet, "D"+strconv.Itoa(r), row.IssuedByUserName)
		_ = f.SetCellValue(sheet, "E"+strconv.Itoa(r), row.DepartmentName)
		_ = f.SetCellValue(sheet, "F"+strconv.Itoa(r), row.ItemName)
		_ = f.SetCellValue(sheet, "G"+strconv.Itoa(r), row.Quantity)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to build export")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="transactions_export.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func (h *TransactionsHandler) IssuePage(c *gin.Context) {
	issueForm := IssueFormData{Quantity: "1"}
	if warehouseUserID, _, err := h.currentWarehouseActor(c); err == nil {
		issueForm.IssuedByUserID = warehouseUserID
	}
	h.renderIssuePage(c, http.StatusOK, issueForm, "", "")
}

func (h *TransactionsHandler) ReturnPage(c *gin.Context) {
	returnForm := ReturnFormData{QuantityReturned: "1"}
	if warehouseUserID, _, err := h.currentWarehouseActor(c); err == nil {
		returnForm.ReceivedByUserID = warehouseUserID
	}
	h.renderReturnPage(c, http.StatusOK, returnForm, "", "")
}

func (h *TransactionsHandler) Issue(c *gin.Context) {
	issueForm := IssueFormData{
		IssuedToUserName: c.PostForm("issued_to_user_name"),
		IssuedByUserID: c.PostForm("issued_by_user_id"),
		ItemID:         c.PostForm("item_id"),
		Quantity:       c.PostForm("quantity"),
	}
	if issueForm.Quantity == "" {
		issueForm.Quantity = "1"
	}

	qty, err := strconv.Atoi(issueForm.Quantity)
	if err != nil {
		h.renderIssuePage(c, http.StatusBadRequest, issueForm, "quantity must be a number", "")
		return
	}

	lang := getLang(c)
	t := translator(lang)
	if strings.TrimSpace(issueForm.IssuedToUserName) == "" {
		h.renderIssuePage(c, http.StatusBadRequest, issueForm, t("issued_to_required"), "")
		return
	}
	issuedToUserID, ok := h.tryResolveUserIDByName(issueForm.IssuedToUserName)
	if !ok {
		h.renderIssuePage(c, http.StatusBadRequest, issueForm, t("employee_not_in_list_hint"), "")
		return
	}
	warehouseUserID, _, actorErr := h.currentWarehouseActor(c)
	if actorErr != nil {
		h.renderIssuePage(c, http.StatusBadRequest, issueForm, actorErr.Error(), "")
		return
	}
	issueForm.IssuedByUserID = warehouseUserID
	issueForm.IssuedToUserID = issuedToUserID

	_, err = h.txSvc.IssueItem(issueForm.ItemID, qty, issuedToUserID, warehouseUserID)
	if err != nil {
		h.renderIssuePage(c, http.StatusBadRequest, issueForm, localizeIssueError(getLang(c), err.Error()), "")
		return
	}

	empty := IssueFormData{Quantity: "1"}
	h.renderIssuePage(c, http.StatusOK, empty, "", translator(getLang(c))("issue_success"))
}

func (h *TransactionsHandler) Return(c *gin.Context) {
	transactionID := c.PostForm("transaction_id")
	qtyStr := c.PostForm("quantity")
	returnedByUserName := c.PostForm("returned_by_user_name")
	receivedByUserID := c.PostForm("received_by_user_id")

	qty, err := strconv.Atoi(qtyStr)
	if err != nil {
		returnForm := ReturnFormData{
			TransactionID:    transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned: qtyStr,
			ReceivedByUserID: receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, "failed to parse quantity", "")
		return
	}

	_, ok, err := h.txSvc.GetByID(transactionID)
	if err != nil {
		returnForm := ReturnFormData{
			TransactionID:    transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned: qtyStr,
			ReceivedByUserID: receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, err.Error(), "")
		return
	}
	if !ok {
		returnForm := ReturnFormData{
			TransactionID:    transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned: qtyStr,
			ReceivedByUserID: receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, "transaction not found", "")
		return
	}

	lang := getLang(c)
	t := translator(lang)
	if strings.TrimSpace(returnedByUserName) == "" {
		returnForm := ReturnFormData{
			TransactionID:      transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned:   qtyStr,
			ReceivedByUserID:   receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, t("returned_by_required"), "")
		return
	}
	returnedByUserID, ok := h.tryResolveUserIDByName(returnedByUserName)
	if !ok {
		returnForm := ReturnFormData{
			TransactionID:      transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned:   qtyStr,
			ReceivedByUserID:   receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, t("employee_not_in_list_hint"), "")
		return
	}
	warehouseUserID, _, actorErr := h.currentWarehouseActor(c)
	if actorErr != nil {
		returnForm := ReturnFormData{
			TransactionID:      transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned:   qtyStr,
			ReceivedByUserID:   receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, actorErr.Error(), "")
		return
	}
	receivedByUserID = warehouseUserID

	_, err = h.txSvc.ReturnItem(transactionID, qty, returnedByUserID, receivedByUserID)
	if err != nil {
		returnForm := ReturnFormData{
			TransactionID:    transactionID,
			ReturnedByUserName: returnedByUserName,
			QuantityReturned: qtyStr,
			ReceivedByUserID: receivedByUserID,
		}
		h.renderReturnPage(c, http.StatusBadRequest, returnForm, err.Error(), "")
		return
	}

	returnForm := ReturnFormData{QuantityReturned: "1"}
	h.renderReturnPage(c, http.StatusOK, returnForm, "", translator(getLang(c))("return_success"))
}

func (h *TransactionsHandler) renderIssuePage(c *gin.Context, status int, issueForm IssueFormData, errMsg string, success string) {
	lang := getLang(c)
	tfunc := translator(lang)
	warehouseUserID, warehouseUserName, _ := h.currentWarehouseActor(c)
	if issueForm.IssuedByUserID == "" {
		issueForm.IssuedByUserID = warehouseUserID
	}

	data, renderErr := h.buildDashboardPageData(lang, issueForm, ReturnFormData{QuantityReturned: "1"}, errMsg, success, "", warehouseUserID, warehouseUserName)
	if renderErr != nil {
		c.String(http.StatusInternalServerError, "failed to render issue page")
		return
	}

	c.HTML(status, "issue.html", PageData{
		Lang:    lang,
		T:       tfunc,
		Success: success,
		Error:   errMsg,
		Data:    data,
	})
}

func (h *TransactionsHandler) renderReturnPage(c *gin.Context, status int, returnForm ReturnFormData, errMsg string, success string) {
	lang := getLang(c)
	tfunc := translator(lang)
	warehouseUserID, warehouseUserName, _ := h.currentWarehouseActor(c)
	if returnForm.ReceivedByUserID == "" {
		returnForm.ReceivedByUserID = warehouseUserID
	}

	data, renderErr := h.buildDashboardPageData(lang, IssueFormData{Quantity: "1"}, returnForm, errMsg, success, "", warehouseUserID, warehouseUserName)
	if renderErr != nil {
		c.String(http.StatusInternalServerError, "failed to render return page")
		return
	}

	c.HTML(status, "return.html", PageData{
		Lang:    lang,
		T:       tfunc,
		Success: success,
		Error:   errMsg,
		Data:    data,
	})
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
		deptName := toUser.DepartmentID
		if d, ok := deptMap[toUser.DepartmentID]; ok && d.Name != "" {
			deptName = d.Name
		}
		if deptName == "" {
			deptName = tx.DepartmentID
		}
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
			IssuedToUserID:   tx.IssuedToUserID,
			IssuedToUserName: toName,
			IssuedByUserName: byName,
			DepartmentName:   deptName,
		})
	}

	for _, ret := range rets {
		toUser := userMap[ret.ReturnedByUserID]
		byUser := userMap[ret.ReceivedByUserID]
		deptName := toUser.DepartmentID
		if d, ok := deptMap[toUser.DepartmentID]; ok && d.Name != "" {
			deptName = d.Name
		}
		if deptName == "" {
			deptName = ret.DepartmentID
		}

		issueTx := issueByID[ret.TransactionID]
		itemName := issueTx.ItemName

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
			ItemName:         itemName,
			Quantity:         ret.QuantityReturned,
			IssuedToUserID:   ret.ReturnedByUserID,
			IssuedToUserName: toName,
			IssuedByUserName: byName,
			DepartmentName:   deptName,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Timestamp > rows[j].Timestamp
	})

	return txs, rows, nil
}

func (h *TransactionsHandler) buildDashboardPageData(lang string, issueForm IssueFormData, returnForm ReturnFormData, errMsg string, flash string, flashError string, warehouseUserID string, warehouseUserName string) (DashboardPageData, error) {
	tfunc := translator(lang)
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
		CurrentWarehouseUserID:   warehouseUserID,
		CurrentWarehouseUserName: warehouseUserName,
		ReturnIssueOptions: returnOptions,
		Error:               errMsg,
		IssueForm:           issueForm,
		ReturnForm:         returnForm,
		TransactionsTableData: TransactionsTableData{
			Lang:         lang,
			T:            tfunc,
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

var stockMsgRe = regexp.MustCompile(`Available:\s*(\d+),\s*requested:\s*(\d+)`)

func localizeIssueError(lang string, raw string) string {
	if strings.Contains(raw, "Not enough stock.") {
		m := stockMsgRe.FindStringSubmatch(raw)
		if len(m) == 3 {
			available, _ := strconv.Atoi(m[1])
			requested, _ := strconv.Atoi(m[2])
			return i18n.Tf(lang, "error_stock_detailed", available, requested)
		}
		return i18n.T(lang, "error_stock")
	}
	return raw
}

func (h *TransactionsHandler) tryResolveUserIDByName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	users, err := h.usersSvc.List()
	if err != nil {
		return "", false
	}
	for _, u := range users {
		if strings.EqualFold(strings.TrimSpace(u.Name), name) {
			return u.ID, true
		}
	}
	return "", false
}

func (h *TransactionsHandler) defaultDepartmentID() (string, error) {
	depts, err := h.deptsSvc.List()
	if err != nil {
		return "", err
	}
	if len(depts) == 0 {
		return "", errors.New("no departments configured")
	}
	return depts[0].ID, nil
}

func (h *TransactionsHandler) currentWarehouseActor(c *gin.Context) (string, string, error) {
	accountID, ok := currentAccountID(c)
	if !ok {
		return "", "", errors.New("auth account not found")
	}
	account, ok, err := h.authSvc.GetByID(accountID)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", errors.New("auth account not found")
	}
	userID, err := h.resolveOrCreateWarehouseUserByName(account.Name)
	if err != nil {
		return "", "", err
	}
	return userID, account.Name, nil
}

func (h *TransactionsHandler) resolveOrCreateWarehouseUserByName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("warehouse user name is empty")
	}
	users, err := h.usersSvc.List()
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if strings.EqualFold(strings.TrimSpace(u.Name), name) {
			if strings.EqualFold(strings.TrimSpace(u.Role), "warehouse") {
				return u.ID, nil
			}
			return "", errors.New("current user is not warehouse staff")
		}
	}
	deptID, err := h.defaultDepartmentID()
	if err != nil {
		return "", err
	}
	autoEmployeeID := "AUTO-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	u, err := h.usersSvc.AddUser(autoEmployeeID, name, deptID, "warehouse")
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

