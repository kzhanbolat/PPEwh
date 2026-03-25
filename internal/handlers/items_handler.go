package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

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
	items, err := h.itemsSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "items.html", ItemsTableData{Error: "failed to load items"})
		return
	}

	c.HTML(http.StatusOK, "items.html", ItemsTableData{
		Items: items,
	})
}

func (h *ItemsHandler) Add(c *gin.Context) {
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
		c.HTML(http.StatusBadRequest, "items_table.html", ItemsTableData{Items: items, Error: "quantity must be a number"})
		return
	}

	_, err = h.itemsSvc.AddItem(name, size, qty, issueDate, expiryDate)
	items, listErr := h.itemsSvc.List()
	if listErr != nil {
		c.HTML(http.StatusInternalServerError, "items_table.html", ItemsTableData{Items: items, Error: "failed to reload items"})
		return
	}

	if err != nil {
		c.HTML(http.StatusBadRequest, "items_table.html", ItemsTableData{Items: items, Error: err.Error()})
		return
	}

	c.HTML(http.StatusOK, "items_table.html", ItemsTableData{Items: items, Success: "Item added successfully."})
}

