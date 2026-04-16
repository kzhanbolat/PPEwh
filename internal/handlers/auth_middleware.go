package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ppewh/internal/services"
)

func RequireAuth(authSvc *services.AuthService, sessionSvc *services.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := getLang(c)
		token, err := c.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(token) == "" {
			c.Redirect(http.StatusSeeOther, "/login?lang="+lang)
			c.Abort()
			return
		}

		accountID, ok := sessionSvc.GetAccountID(token)
		if !ok {
			c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusSeeOther, "/login?lang="+lang)
			c.Abort()
			return
		}

		account, exists, err := authSvc.GetByID(accountID)
		if err != nil || !exists {
			c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusSeeOther, "/login?lang="+lang)
			c.Abort()
			return
		}
		if !account.IsAdmin && !account.IsApproved {
			c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusSeeOther, "/login?lang="+lang)
			c.Abort()
			return
		}
		if account.MustResetPassword && c.FullPath() != "/change-password" {
			c.Redirect(http.StatusSeeOther, "/change-password?lang="+lang)
			c.Abort()
			return
		}

		c.Set("auth_account_id", account.ID)
		c.Set("auth_is_admin", account.IsAdmin)
		c.Set("auth_name", account.Name)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := getLang(c)
		t := translator(lang)
		renderForbidden := func() {
			c.HTML(http.StatusForbidden, "error.html", PageData{
				Lang:  lang,
				T:     t,
				Error: t("access_forbidden"),
				Data: map[string]string{
					"Title": t("forbidden"),
				},
			})
			c.Abort()
		}

		v, ok := c.Get("auth_is_admin")
		if !ok {
			renderForbidden()
			return
		}
		isAdmin, ok := v.(bool)
		if !ok || !isAdmin {
			renderForbidden()
			return
		}
		c.Next()
	}
}
