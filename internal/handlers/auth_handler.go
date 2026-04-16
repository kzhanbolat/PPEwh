package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ppewh/internal/services"
)

const sessionCookieName = "ppe_session"

type AuthHandler struct {
	authSvc    *services.AuthService
	sessionSvc *services.SessionService
	usersSvc   *services.UsersService
	deptsSvc   *services.DepartmentsService
}

func NewAuthHandler(authSvc *services.AuthService, sessionSvc *services.SessionService, usersSvc *services.UsersService, deptsSvc *services.DepartmentsService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, sessionSvc: sessionSvc, usersSvc: usersSvc, deptsSvc: deptsSvc}
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	c.HTML(http.StatusOK, "auth_login.html", AuthLoginPageData{
		Lang: lang,
		T:    t,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")

	account, err := h.authSvc.Authenticate(email, password)
	if err != nil {
		c.HTML(http.StatusBadRequest, "auth_login.html", AuthLoginPageData{
			Lang:  lang,
			T:     t,
			Email: email,
			Error: err.Error(),
		})
		return
	}

	token, expiresAt, err := h.sessionSvc.Create(account.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "auth_login.html", AuthLoginPageData{
			Lang:  lang,
			T:     t,
			Email: email,
			Error: "failed to start session",
		})
		return
	}

	c.SetCookie(sessionCookieName, token, int(expiresAt.Sub(time.Now()).Seconds()), "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/?lang="+lang)
}

func (h *AuthHandler) RegisterPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	c.HTML(http.StatusOK, "auth_register.html", AuthRegisterPageData{
		Lang: lang,
		T:    t,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	isWarehouse := c.PostForm("is_warehouse_worker") == "on"

	err := h.authSvc.Register(name, email, password, isWarehouse)
	if err != nil {
		c.HTML(http.StatusBadRequest, "auth_register.html", AuthRegisterPageData{
			Lang:              lang,
			T:                 t,
			Name:              name,
			Email:             email,
			IsWarehouseWorker: isWarehouse,
			Error:             err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "auth_login.html", AuthLoginPageData{
		Lang:    lang,
		T:       t,
		Email:   email,
		Success: t("registration_success_pending"),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	lang := getLang(c)
	token, err := c.Cookie(sessionCookieName)
	if err == nil && strings.TrimSpace(token) != "" {
		h.sessionSvc.Delete(token)
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/login?lang="+lang)
}

func (h *AuthHandler) ChangePasswordPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	c.HTML(http.StatusOK, "auth_change_password.html", AuthChangePasswordPageData{
		Lang: lang,
		T:    t,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)

	accountID, ok := currentAccountID(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login?lang="+lang)
		return
	}

	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	err := h.authSvc.ChangePassword(accountID, currentPassword, newPassword)
	if err != nil {
		c.HTML(http.StatusBadRequest, "auth_change_password.html", AuthChangePasswordPageData{
			Lang:  lang,
			T:     t,
			Error: err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "auth_change_password.html", AuthChangePasswordPageData{
		Lang:    lang,
		T:       t,
		Success: t("password_changed"),
	})
}

func (h *AuthHandler) AdminAccessPage(c *gin.Context) {
	lang := getLang(c)
	t := translator(lang)
	accounts, err := h.authSvc.ListAccounts()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "admin_access.html", AdminAccessPageData{
			Lang:  lang,
			T:     t,
			Error: "failed to load accounts",
		})
		return
	}
	c.HTML(http.StatusOK, "admin_access.html", AdminAccessPageData{
		Lang:     lang,
		T:        t,
		Accounts: accounts,
	})
}

func (h *AuthHandler) AdminSetApproval(c *gin.Context) {
	lang := getLang(c)
	accountID := strings.TrimSpace(c.PostForm("account_id"))
	approved := c.PostForm("approved") == "on"
	if err := h.authSvc.SetApproval(accountID, approved); err != nil {
		h.renderAdminPageWithMessage(c, http.StatusBadRequest, "", err.Error())
		return
	}
	if approved {
		if err := h.ensureApprovedAccountInEmployees(accountID); err != nil {
			h.renderAdminPageWithMessage(c, http.StatusBadRequest, "", err.Error())
			return
		}
	}
	c.Redirect(http.StatusSeeOther, "/admin/access?lang="+lang)
}

func (h *AuthHandler) AdminResetPassword(c *gin.Context) {
	lang := getLang(c)
	accountID := strings.TrimSpace(c.PostForm("account_id"))
	newPassword := strings.TrimSpace(c.PostForm("new_password"))
	if err := h.authSvc.ResetPasswordByAdmin(accountID, newPassword); err != nil {
		h.renderAdminPageWithMessage(c, http.StatusBadRequest, "", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/access?lang="+lang)
}

func (h *AuthHandler) renderAdminPageWithMessage(c *gin.Context, status int, success string, errMsg string) {
	lang := getLang(c)
	t := translator(lang)
	accounts, _ := h.authSvc.ListAccounts()
	c.HTML(status, "admin_access.html", AdminAccessPageData{
		Lang:     lang,
		T:        t,
		Accounts: accounts,
		Success:  success,
		Error:    errMsg,
	})
}

func currentAccountID(c *gin.Context) (string, bool) {
	v, ok := c.Get("auth_account_id")
	if !ok {
		return "", false
	}
	id, ok := v.(string)
	if !ok || strings.TrimSpace(id) == "" {
		return "", false
	}
	return id, true
}

func (h *AuthHandler) ensureApprovedAccountInEmployees(accountID string) error {
	account, ok, err := h.authSvc.GetByID(accountID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("account not found")
	}
	if strings.TrimSpace(account.Name) == "" {
		return errors.New("account name is required")
	}

	users, err := h.usersSvc.List()
	if err != nil {
		return err
	}
	for _, u := range users {
		if strings.EqualFold(strings.TrimSpace(u.Name), strings.TrimSpace(account.Name)) {
			return nil
		}
	}

	deptID, err := h.defaultDepartmentID()
	if err != nil {
		return err
	}
	_, err = h.usersSvc.AddUser(account.Name, deptID, "warehouse")
	return err
}

func (h *AuthHandler) defaultDepartmentID() (string, error) {
	depts, err := h.deptsSvc.List()
	if err != nil {
		return "", err
	}
	if len(depts) == 0 {
		return "", errors.New("no departments configured")
	}
	return depts[0].ID, nil
}

