package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"ppewh/internal/services"
)

type ItemsHandler struct {
	itemsSvc *services.ItemsService
	txSvc    *services.TransactionsService
}

func NewItemsHandler(itemsSvc *services.ItemsService, txSvc *services.TransactionsService, _ any) *ItemsHandler {
	return &ItemsHandler{itemsSvc: itemsSvc, txSvc: txSvc}
}

func (h *ItemsHandler) ListPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	items, err := h.itemsSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "items.html", ItemsTableData{Lang: lang, T: t, Error: "failed to load items"})
		return
	}

	c.HTML(http.StatusOK, "items.html", ItemsTableData{
		Lang:  lang,
		T:     t,
		Items: items,
	})
}

func (h *ItemsHandler) Add(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	name := c.PostForm("name")
	size := c.PostForm("size")
	qtyStr := c.PostForm("quantity")
	issueDate := c.PostForm("issue_date")
	expiryDate := c.PostForm("expiry_date")

	qty, err := strconv.Atoi(qtyStr)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		// For HTMX, respond with a partial so the page doesn't hard reload.
		items, _ := h.itemsSvc.List()
		c.HTML(http.StatusBadRequest, "items_table.html", ItemsTableData{Lang: lang, T: t, Items: items, Error: "quantity must be a number"})
		return
	}

	_, err = h.itemsSvc.AddItem(name, size, qty, issueDate, expiryDate)
	items, listErr := h.itemsSvc.List()
	if listErr != nil {
		c.HTML(http.StatusInternalServerError, "items_table.html", ItemsTableData{Lang: lang, T: t, Items: items, Error: "failed to reload items"})
		return
	}

	if err != nil {
		c.HTML(http.StatusBadRequest, "items_table.html", ItemsTableData{Lang: lang, T: t, Items: items, Error: err.Error()})
		return
	}

	c.HTML(http.StatusOK, "items_table.html", ItemsTableData{Lang: lang, T: t, Items: items, Success: t("item_added_success")})
}

func (h *ItemsHandler) Export(c *gin.Context) {
	items, err := h.itemsSvc.List()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load items")
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	headers := []string{"id", "name", "size", "quantity", "issue_date", "expiry_date"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, it := range items {
		row := i + 2
		_ = f.SetCellValue(sheet, "A"+strconv.Itoa(row), it.ID)
		_ = f.SetCellValue(sheet, "B"+strconv.Itoa(row), it.Name)
		_ = f.SetCellValue(sheet, "C"+strconv.Itoa(row), it.Size)
		_ = f.SetCellValue(sheet, "D"+strconv.Itoa(row), it.Quantity)
		_ = f.SetCellValue(sheet, "E"+strconv.Itoa(row), it.IssueDate)
		_ = f.SetCellValue(sheet, "F"+strconv.Itoa(row), it.ExpiryDate)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to build export")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="items_export.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func (h *ItemsHandler) UploadExcel(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)

	fileHeader, err := c.FormFile("items_file")
	if err != nil {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", t("items_upload_file_required"))
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", t("items_upload_read_failed"))
		return
	}
	defer src.Close()

	wb, err := excelize.OpenReader(src)
	if err != nil {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", t("items_upload_invalid_excel"))
		return
	}
	defer func() { _ = wb.Close() }()

	sheets := wb.GetSheetList()
	if len(sheets) == 0 {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", t("items_upload_empty_sheet"))
		return
	}
	rows, err := wb.GetRows(sheets[0])
	if err != nil || len(rows) == 0 {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", t("items_upload_empty_sheet"))
		return
	}

	headerIndex := map[string]int{}
	for idx, hname := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(hname))
		headerIndex[key] = idx
	}

	nameIdx, okName := headerIndex["name"]
	sizeIdx, okSize := headerIndex["size"]
	qtyIdx, okQty := headerIndex["quantity"]
	issueIdx, okIssue := headerIndex["issue_date"]
	expiryIdx, okExpiry := headerIndex["expiry_date"]
	if !okName || !okSize || !okQty || !okIssue || !okExpiry {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", t("items_upload_missing_columns"))
		return
	}

	var errorsList []string
	added := 0
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		name := valueAt(row, nameIdx)
		size := valueAt(row, sizeIdx)
		qtyStr := valueAt(row, qtyIdx)
		issueDate := valueAt(row, issueIdx)
		expiryDate := valueAt(row, expiryIdx)

		if name == "" && size == "" && qtyStr == "" && issueDate == "" && expiryDate == "" {
			continue
		}
		if name == "" || size == "" || qtyStr == "" || issueDate == "" || expiryDate == "" {
			errorsList = append(errorsList, fmt.Sprintf("%s %d: %s", t("row"), i+1, t("items_upload_required_cells")))
			continue
		}
		qty, err := strconv.Atoi(qtyStr)
		if err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s %d: %s", t("row"), i+1, t("quantity_must_be_number")))
			continue
		}
		if _, err := h.itemsSvc.AddItem(name, size, qty, issueDate, expiryDate); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s %d: %s", t("row"), i+1, err.Error()))
			continue
		}
		added++
	}

	if len(errorsList) > 0 {
		h.renderItemsPage(c, http.StatusBadRequest, lang, t, "", strings.Join(errorsList, "; "))
		return
	}
	success := fmt.Sprintf("%s: %d", t("items_upload_success"), added)
	h.renderItemsPage(c, http.StatusOK, lang, t, success, "")
}

func (h *ItemsHandler) renderItemsPage(c *gin.Context, status int, lang string, t func(string) string, success string, errMsg string) {
	items, err := h.itemsSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "items.html", ItemsTableData{
			Lang:  lang,
			T:     t,
			Error: "failed to load items",
		})
		return
	}
	c.HTML(status, "items.html", ItemsTableData{
		Lang:    lang,
		T:       t,
		Items:   items,
		Success: success,
		Error:   errMsg,
	})
}

