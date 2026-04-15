package handlers

import (
	"bytes"
	"encoding/csv"
	"net/http"

	"github.com/gin-gonic/gin"

	"ppewh/internal/services"
)

type UsersHandler struct {
	usersSvc *services.UsersService
}

func NewUsersHandler(usersSvc *services.UsersService, _ any) *UsersHandler {
	return &UsersHandler{usersSvc: usersSvc}
}

func (h *UsersHandler) ListPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	users, err := h.usersSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "employees.html", UsersTableData{Lang: lang, T: t, Error: "failed to load users"})
		return
	}
	c.HTML(http.StatusOK, "employees.html", UsersTableData{Lang: lang, T: t, Users: users})
}

func (h *UsersHandler) Add(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	name := c.PostForm("name")
	departmentID := c.PostForm("department_id")
	role := c.PostForm("role")

	_, err := h.usersSvc.AddUser(name, departmentID, role)
	users, listErr := h.usersSvc.List()
	if listErr != nil {
		c.HTML(http.StatusInternalServerError, "users_table.html", UsersTableData{Lang: lang, T: t, Users: users, Error: "failed to reload users"})
		return
	}

	if err != nil {
		c.HTML(http.StatusBadRequest, "users_table.html", UsersTableData{Lang: lang, T: t, Users: users, Error: err.Error()})
		return
	}

	c.HTML(http.StatusOK, "users_table.html", UsersTableData{Lang: lang, T: t, Users: users, Success: t("user_added_success")})
}

func (h *UsersHandler) Export(c *gin.Context) {
	users, err := h.usersSvc.List()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load employees")
		return
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "name", "department_id", "role"})
	for _, u := range users {
		_ = w.Write([]string{u.ID, u.Name, u.DepartmentID, u.Role})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		c.String(http.StatusInternalServerError, "failed to build export")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="employees_export.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

