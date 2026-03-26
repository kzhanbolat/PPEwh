package app

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"ppewh/internal/handlers"
	"ppewh/internal/i18n"
	"ppewh/internal/services"
	"ppewh/internal/storage"
)

// Run starts the HTTP server.
//
// Usage:
//   go run main.go
func Run() {
	rootDir, _ := os.Getwd()
	dataDir := filepath.Join(rootDir, "data")

	// Storage -> Services -> Handlers
	if err := i18n.Load(filepath.Join(rootDir, "locales")); err != nil {
		panic(err)
	}

	csvStore, err := storage.NewCSVStore(dataDir)
	if err != nil {
		panic(err)
	}

	itemsSvc := services.NewItemsService(csvStore.Items())
	usersSvc := services.NewUsersService(csvStore.Users())
	deptsSvc := services.NewDepartmentsService(csvStore.Departments())
	txSvc := services.NewTransactionsService(csvStore.Items(), csvStore.Users(), csvStore.Transactions(), csvStore.Returns())

	templates := loadTemplates()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.SetHTMLTemplate(templates)
	r.Static("/static", filepath.Join(rootDir, "static"))

	// Home dashboard
	homeHandler := handlers.NewHomeHandler(itemsSvc, usersSvc, txSvc, deptsSvc)
	r.GET("/", homeHandler.Dashboard)

	// Items
	itemsHandler := handlers.NewItemsHandler(itemsSvc, txSvc, templates)
	r.GET("/items", itemsHandler.ListPage)
	r.POST("/items", itemsHandler.Add)

	// Users
	usersHandler := handlers.NewUsersHandler(usersSvc, templates)
	r.GET("/users", usersHandler.ListPage)
	r.POST("/users", usersHandler.Add)

	// Transactions
	txHandler := handlers.NewTransactionsHandler(itemsSvc, txSvc, usersSvc, deptsSvc, templates)
	r.GET("/transactions", txHandler.History)
	r.POST("/issue", txHandler.Issue)
	r.POST("/return", txHandler.Return)

	// Basic health check
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	_ = r.Run(":" + addr)
}

func loadTemplates() *template.Template {
	// Gin will render named templates. We load all page templates + partial templates.
	t := template.New("").Funcs(template.FuncMap{
		"today": func() string { return services.TodayYYYYMMDD() },
	})

	// Parse full HTML pages
	paths := []string{
		"templates/index.html",
		"templates/items.html",
		"templates/users.html",
		"templates/transactions.html",
		"templates/partials/dashboard_root.html",
		"templates/partials/items_table.html",
		"templates/partials/users_table.html",
		"templates/partials/transactions_table.html",
		"templates/partials/dashboard_transactions.html",
	}

	// Parse all templates into the same template set so `{{template "..."}}` works.
	template.Must(t.ParseFiles(paths...))
	return t
}

