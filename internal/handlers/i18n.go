package handlers

import (
	"github.com/gin-gonic/gin"

	"ppewh/internal/i18n"
)

func getLang(c *gin.Context) string {
	lang := c.Query("lang")
	if lang == "" {
		lang = c.PostForm("lang")
	}
	return i18n.NormalizeLang(lang)
}

func translator(lang string) func(string) string {
	return func(key string) string {
		return i18n.T(lang, key)
	}
}

