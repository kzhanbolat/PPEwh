package handlers

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ppewh/internal/models"
	"ppewh/internal/services"
)

type DashboardHandler struct {
	itemsSvc *services.ItemsService
	txSvc    *services.TransactionsService
	usersSvc *services.UsersService
	deptsSvc *services.DepartmentsService
}

type DashboardSummaryData struct {
	TotalStock         int
	TotalIssued        int
	TotalReturned      int
	LowStockItems      []models.Item
	ExpiringItems      []models.Item
	RecentTransactions []DashboardRecentTransaction
	LowStockThreshold  int
	ExpiringWithinDays int

	// Filters
	FromDate           string
	ToDate             string
	SelectedEmployeeID string
	SelectedDepartmentID string
	SelectedItemID     string
	Employees          []models.User
	Departments        []models.Department
	Items              []models.Item

	// Chart series
	LineLabels     []string
	IssuedCounts   []int
	ReturnedCounts []int
	BarLabels      []string
	ItemQuantities []int

	DeptLabels            []string
	DeptUsageDatasets     []DashboardChartDataset
}

type DashboardRecentTransaction struct {
	ItemName  string
	EventType string
	UserName  string
	Timestamp string
}

type dashboardEvent struct {
	ItemID    string
	ItemName  string
	EventType string
	Quantity  int
	UserID    string
	UserName  string
	Timestamp string
	DateKey   string
}

type DashboardChartDataset struct {
	Label           string
	Data            []int
	BackgroundColor string
	BorderColor     string
	BorderWidth     int
}

func NewDashboardHandler(itemsSvc *services.ItemsService, txSvc *services.TransactionsService, usersSvc *services.UsersService) *DashboardHandler {
	return &DashboardHandler{
		itemsSvc: itemsSvc,
		txSvc:    txSvc,
		usersSvc: usersSvc,
	}
}

func NewDashboardHandlerWithDepartments(itemsSvc *services.ItemsService, txSvc *services.TransactionsService, usersSvc *services.UsersService, deptsSvc *services.DepartmentsService) *DashboardHandler {
	h := NewDashboardHandler(itemsSvc, txSvc, usersSvc)
	h.deptsSvc = deptsSvc
	return h
}

func (h *DashboardHandler) Dashboard(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)

	const lowStockThreshold = 10
	const expiringWithinDays = 30
	fromDate := strings.TrimSpace(c.Query("from"))
	toDate := strings.TrimSpace(c.Query("to"))
	selectedEmployeeID := strings.TrimSpace(c.Query("employee_id"))
	selectedDepartmentID := strings.TrimSpace(c.Query("department_id"))
	selectedItemID := strings.TrimSpace(c.Query("item_id"))

	items, err := h.itemsSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", PageData{
			Lang:  lang,
			T:     t,
			Error: "failed to load items",
			Data:  DashboardSummaryData{},
		})
		return
	}
	txs, err := h.txSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", PageData{
			Lang:  lang,
			T:     t,
			Error: "failed to load transactions",
			Data:  DashboardSummaryData{},
		})
		return
	}
	rets, err := h.txSvc.ListReturns()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", PageData{
			Lang:  lang,
			T:     t,
			Error: "failed to load returns",
			Data:  DashboardSummaryData{},
		})
		return
	}
	users, err := h.usersSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", PageData{
			Lang:  lang,
			T:     t,
			Error: "failed to load users",
			Data:  DashboardSummaryData{},
		})
		return
	}
	deptMap := map[string]models.Department{}
	departments := make([]models.Department, 0)
	if h.deptsSvc != nil {
		depts, deptErr := h.deptsSvc.List()
		if deptErr == nil {
			departments = depts
			for _, d := range depts {
				deptMap[d.ID] = d
			}
		}
	}

	userMap := map[string]models.User{}
	for _, u := range users {
		userMap[u.ID] = u
	}
	issueMap := map[string]models.Transaction{}
	for _, tx := range txs {
		issueMap[tx.ID] = tx
	}

	events := make([]dashboardEvent, 0, len(txs)+len(rets))
	totalIssued := 0
	totalReturned := 0

	for _, tx := range txs {
		txDate := dateKey(tx.Timestamp)
		if !withinDateRange(txDate, fromDate, toDate) {
			continue
		}
		if selectedEmployeeID != "" && tx.IssuedToUserID != selectedEmployeeID {
			continue
		}
		toUser := userMap[tx.IssuedToUserID]
		if selectedDepartmentID != "" && toUser.DepartmentID != selectedDepartmentID {
			continue
		}
		if selectedItemID != "" && tx.ItemID != selectedItemID {
			continue
		}
		totalIssued += tx.Quantity
		events = append(events, dashboardEvent{
			ItemID:    tx.ItemID,
			ItemName:  tx.ItemName,
			EventType: "ISSUE",
			Quantity:  tx.Quantity,
			UserID:    tx.IssuedToUserID,
			UserName:  toUser.Name,
			Timestamp: tx.Timestamp,
			DateKey:   txDate,
		})
	}

	for _, ret := range rets {
		retDate := dateKey(ret.Timestamp)
		if !withinDateRange(retDate, fromDate, toDate) {
			continue
		}
		if selectedEmployeeID != "" && ret.ReturnedByUserID != selectedEmployeeID {
			continue
		}
		returnedBy := userMap[ret.ReturnedByUserID]
		if selectedDepartmentID != "" && returnedBy.DepartmentID != selectedDepartmentID {
			continue
		}
		issueTx := issueMap[ret.TransactionID]
		itemID := issueTx.ItemID
		if itemID == "" {
			itemID = ret.ItemID
		}
		if selectedItemID != "" && itemID != selectedItemID {
			continue
		}
		totalReturned += ret.QuantityReturned
		events = append(events, dashboardEvent{
			ItemID:    itemID,
			ItemName:  issueTx.ItemName,
			EventType: "RETURN",
			Quantity:  ret.QuantityReturned,
			UserID:    ret.ReturnedByUserID,
			UserName:  returnedBy.Name,
			Timestamp: ret.Timestamp,
			DateKey:   retDate,
		})
	}

	lineIssued := map[string]int{}
	lineReturned := map[string]int{}
	barItemQty := map[string]int{}
	deptItemIssued := map[string]map[string]int{}
	filteredItemIDs := map[string]bool{}
	for _, ev := range events {
		if ev.ItemID != "" {
			filteredItemIDs[ev.ItemID] = true
		}
		if ev.EventType == "ISSUE" {
			lineIssued[ev.DateKey] += 1
			u := userMap[ev.UserID]
			deptName := u.DepartmentID
			if d, ok := deptMap[u.DepartmentID]; ok && d.Name != "" {
				deptName = d.Name
			}
			if deptName == "" {
				deptName = "N/A"
			}
			if _, ok := deptItemIssued[deptName]; !ok {
				deptItemIssued[deptName] = map[string]int{}
			}
			deptItemIssued[deptName][ev.ItemName] += ev.Quantity
		} else {
			lineReturned[ev.DateKey] += 1
		}
		// Inventory activity quantity for the selected item or all items.
		barItemQty[ev.DateKey] += ev.Quantity
	}

	// Apply the same dashboard filter scope to stock/low/expiring blocks.
	// - No filters: use all items.
	// - item filter: only selected item.
	// - date/employee/department filters: items present in filtered events.
	hasTxScopeFilter := fromDate != "" || toDate != "" || selectedEmployeeID != "" || selectedDepartmentID != ""
	scopedItems := make([]models.Item, 0, len(items))
	for _, item := range items {
		if selectedItemID != "" && item.ID != selectedItemID {
			continue
		}
		if hasTxScopeFilter && selectedItemID == "" && !filteredItemIDs[item.ID] {
			continue
		}
		scopedItems = append(scopedItems, item)
	}

	totalStock := 0
	lowStockItems := make([]models.Item, 0)
	expiringItems := make([]models.Item, 0)
	seenLowStock := map[string]bool{}
	seenExpiring := map[string]bool{}
	today := time.Now().Truncate(24 * time.Hour)
	expiryLimit := today.AddDate(0, 0, expiringWithinDays)
	for _, item := range scopedItems {
		totalStock += item.Quantity
		lowKey := strings.ToLower(strings.TrimSpace(item.Name))
		if lowKey == "" {
			lowKey = item.ID
		}
		if item.Quantity < lowStockThreshold && !seenLowStock[lowKey] {
			lowStockItems = append(lowStockItems, item)
			seenLowStock[lowKey] = true
		}
		if item.ExpiryDate == "" {
			continue
		}
		expiryDate, parseErr := time.Parse("2006-01-02", strings.TrimSpace(item.ExpiryDate))
		if parseErr != nil {
			continue
		}
		expKey := strings.ToLower(strings.TrimSpace(item.Name))
		if expKey == "" {
			expKey = item.ID
		}
		if (expiryDate.Equal(today) || expiryDate.After(today)) && (expiryDate.Equal(expiryLimit) || expiryDate.Before(expiryLimit)) && !seenExpiring[expKey] {
			expiringItems = append(expiringItems, item)
			seenExpiring[expKey] = true
		}
	}

	lineLabels := make([]string, 0, len(lineIssued)+len(lineReturned))
	lineLabelSeen := map[string]bool{}
	for date := range lineIssued {
		lineLabelSeen[date] = true
		lineLabels = append(lineLabels, date)
	}
	for date := range lineReturned {
		if !lineLabelSeen[date] {
			lineLabels = append(lineLabels, date)
		}
	}
	sort.Strings(lineLabels)
	issuedCounts := make([]int, 0, len(lineLabels))
	returnedCounts := make([]int, 0, len(lineLabels))
	for _, d := range lineLabels {
		issuedCounts = append(issuedCounts, lineIssued[d])
		returnedCounts = append(returnedCounts, lineReturned[d])
	}

	barLabels := make([]string, 0, len(barItemQty))
	for d := range barItemQty {
		barLabels = append(barLabels, d)
	}
	sort.Strings(barLabels)
	itemQuantities := make([]int, 0, len(barLabels))
	for _, d := range barLabels {
		itemQuantities = append(itemQuantities, barItemQty[d])
	}

	deptLabels := make([]string, 0, len(deptItemIssued))
	itemTypeSet := map[string]bool{}
	for deptName, byItem := range deptItemIssued {
		deptLabels = append(deptLabels, deptName)
		for itemName := range byItem {
			itemTypeSet[itemName] = true
		}
	}
	sort.Strings(deptLabels)
	itemTypes := make([]string, 0, len(itemTypeSet))
	for itemName := range itemTypeSet {
		itemTypes = append(itemTypes, itemName)
	}
	sort.Strings(itemTypes)

	palette := []string{
		"rgba(37, 99, 235, 0.7)",
		"rgba(16, 185, 129, 0.7)",
		"rgba(245, 158, 11, 0.7)",
		"rgba(239, 68, 68, 0.7)",
		"rgba(168, 85, 247, 0.7)",
		"rgba(6, 182, 212, 0.7)",
		"rgba(20, 184, 166, 0.7)",
		"rgba(236, 72, 153, 0.7)",
	}
	stackedDatasets := make([]DashboardChartDataset, 0, len(itemTypes))
	for i, itemName := range itemTypes {
		data := make([]int, 0, len(deptLabels))
		for _, deptName := range deptLabels {
			data = append(data, deptItemIssued[deptName][itemName])
		}
		color := palette[i%len(palette)]
		stackedDatasets = append(stackedDatasets, DashboardChartDataset{
			Label:           itemName,
			Data:            data,
			BackgroundColor: color,
			BorderColor:     strings.Replace(color, "0.7", "1", 1),
			BorderWidth:     1,
		})
	}

	recent := make([]DashboardRecentTransaction, 0, len(events))
	for _, ev := range events {
		recent = append(recent, DashboardRecentTransaction{
			ItemName:  ev.ItemName,
			EventType: ev.EventType,
			UserName:  ev.UserName,
			Timestamp: ev.Timestamp,
		})
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].Timestamp > recent[j].Timestamp })
	if len(recent) > 10 {
		recent = recent[:10]
	}

	c.HTML(http.StatusOK, "dashboard.html", PageData{
		Lang: lang,
		T:    t,
		Data: DashboardSummaryData{
			TotalStock:         totalStock,
			TotalIssued:        totalIssued,
			TotalReturned:      totalReturned,
			LowStockItems:      lowStockItems,
			ExpiringItems:      expiringItems,
			RecentTransactions: recent,
			LowStockThreshold:  lowStockThreshold,
			ExpiringWithinDays: expiringWithinDays,
			FromDate:           fromDate,
			ToDate:             toDate,
			SelectedEmployeeID: selectedEmployeeID,
			SelectedDepartmentID: selectedDepartmentID,
			SelectedItemID:     selectedItemID,
			Employees:          users,
			Departments:        departments,
			Items:              items,
			LineLabels:         lineLabels,
			IssuedCounts:       issuedCounts,
			ReturnedCounts:     returnedCounts,
			BarLabels:          barLabels,
			ItemQuantities:     itemQuantities,
			DeptLabels:         deptLabels,
			DeptUsageDatasets:  stackedDatasets,
		},
	})
}

func dateKey(timestamp string) string {
	if len(timestamp) >= 10 {
		return timestamp[:10]
	}
	return timestamp
}

func withinDateRange(date, fromDate, toDate string) bool {
	if date == "" {
		return false
	}
	if fromDate != "" && date < fromDate {
		return false
	}
	if toDate != "" && date > toDate {
		return false
	}
	return true
}
