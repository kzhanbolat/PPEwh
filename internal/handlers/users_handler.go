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

type UsersHandler struct {
	usersSvc *services.UsersService
}

func NewUsersHandler(usersSvc *services.UsersService, _ any) *UsersHandler {
	return &UsersHandler{usersSvc: usersSvc}
}

func (h *UsersHandler) ListPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	isAdmin := false
	if v, ok := c.Get("auth_is_admin"); ok {
		if b, ok := v.(bool); ok {
			isAdmin = b
		}
	}
	users, err := h.usersSvc.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "employees.html", UsersTableData{Lang: lang, T: t, IsAdmin: isAdmin, Error: "failed to load users"})
		return
	}
	c.HTML(http.StatusOK, "employees.html", UsersTableData{Lang: lang, T: t, IsAdmin: isAdmin, Users: users})
}

func (h *UsersHandler) Add(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	employeeID := c.PostForm("employee_id")
	name := c.PostForm("name")
	departmentID := c.PostForm("department_id")
	role := c.PostForm("role")

	_, err := h.usersSvc.AddUser(employeeID, name, departmentID, role)
	users, listErr := h.usersSvc.List()
	if listErr != nil {
		c.HTML(http.StatusInternalServerError, "users_table.html", UsersTableData{Lang: lang, T: t, IsAdmin: true, Users: users, Error: "failed to reload users"})
		return
	}

	if err != nil {
		c.HTML(http.StatusBadRequest, "users_table.html", UsersTableData{Lang: lang, T: t, IsAdmin: true, Users: users, Error: err.Error()})
		return
	}

	c.HTML(http.StatusOK, "users_table.html", UsersTableData{Lang: lang, T: t, IsAdmin: true, Users: users, Success: t("user_added_success")})
}

func (h *UsersHandler) Export(c *gin.Context) {
	users, err := h.usersSvc.List()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load employees")
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	headers := []string{"id", "employee_id", "name", "department_id", "role"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, u := range users {
		row := i + 2
		_ = f.SetCellValue(sheet, "A"+strconv.Itoa(row), u.ID)
		_ = f.SetCellValue(sheet, "B"+strconv.Itoa(row), u.EmployeeID)
		_ = f.SetCellValue(sheet, "C"+strconv.Itoa(row), u.Name)
		_ = f.SetCellValue(sheet, "D"+strconv.Itoa(row), u.DepartmentID)
		_ = f.SetCellValue(sheet, "E"+strconv.Itoa(row), u.Role)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to build export")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="employees_export.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func (h *UsersHandler) UploadExcel(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)

	fileHeader, err := c.FormFile("employees_file")
	if err != nil {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", t("employees_upload_file_required"))
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", t("employees_upload_read_failed"))
		return
	}
	defer src.Close()

	wb, err := excelize.OpenReader(src)
	if err != nil {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", t("employees_upload_invalid_excel"))
		return
	}
	defer func() { _ = wb.Close() }()

	sheets := wb.GetSheetList()
	if len(sheets) == 0 {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", t("employees_upload_empty_sheet"))
		return
	}
	rows, err := wb.GetRows(sheets[0])
	if err != nil || len(rows) == 0 {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", t("employees_upload_empty_sheet"))
		return
	}

	headerIndex := map[string]int{}
	for idx, hname := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(hname))
		headerIndex[key] = idx
	}

	employeeIDIdx, okEmpID := headerIndex["employee_id"]
	nameIdx, okName := headerIndex["name"]
	if !okName {
		nameIdx, okName = headerIndex["full_name"]
	}
	departmentIdx, okDept := headerIndex["department_id"]
	roleIdx, okRole := headerIndex["role"]
	if !okEmpID || !okName || !okDept || !okRole {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", t("employees_upload_missing_columns"))
		return
	}

	var errorsList []string
	added := 0
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		employeeID := valueAt(row, employeeIDIdx)
		name := valueAt(row, nameIdx)
		departmentID := valueAt(row, departmentIdx)
		role := valueAt(row, roleIdx)

		// Skip fully empty rows.
		if employeeID == "" && name == "" && departmentID == "" && role == "" {
			continue
		}
		if employeeID == "" || name == "" || departmentID == "" || role == "" {
			errorsList = append(errorsList, fmt.Sprintf("%s %d: %s", t("row"), i+1, t("employees_upload_required_cells")))
			continue
		}
		if _, err := h.usersSvc.AddUser(employeeID, name, departmentID, role); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s %d: %s", t("row"), i+1, err.Error()))
			continue
		}
		added++
	}

	if len(errorsList) > 0 {
		h.renderEmployeesPage(c, http.StatusBadRequest, lang, t, true, "", strings.Join(errorsList, "; "))
		return
	}

	success := fmt.Sprintf("%s: %d", t("employees_upload_success"), added)
	h.renderEmployeesPage(c, http.StatusOK, lang, t, true, success, "")
}

func (h *UsersHandler) renderUsersTable(c *gin.Context, status int, lang string, t func(string) string, isAdmin bool, success string, errMsg string) {
	users, listErr := h.usersSvc.List()
	if listErr != nil {
		c.HTML(http.StatusInternalServerError, "users_table.html", UsersTableData{
			Lang:    lang,
			T:       t,
			IsAdmin: isAdmin,
			Error:   "failed to load users",
		})
		return
	}
	c.HTML(status, "users_table.html", UsersTableData{
		Lang:    lang,
		T:       t,
		IsAdmin: isAdmin,
		Users:   users,
		Success: success,
		Error:   errMsg,
	})
}

func valueAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func (h *UsersHandler) renderEmployeesPage(c *gin.Context, status int, lang string, t func(string) string, isAdmin bool, success string, errMsg string) {
	users, listErr := h.usersSvc.List()
	if listErr != nil {
		c.HTML(http.StatusInternalServerError, "employees.html", UsersTableData{
			Lang:    lang,
			T:       t,
			IsAdmin: isAdmin,
			Error:   "failed to load users",
		})
		return
	}
	c.HTML(status, "employees.html", UsersTableData{
		Lang:    lang,
		T:       t,
		IsAdmin: isAdmin,
		Users:   users,
		Success: success,
		Error:   errMsg,
	})
}

