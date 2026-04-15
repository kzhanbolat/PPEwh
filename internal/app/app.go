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
	r.StaticFile("/logo.png", filepath.Join(rootDir, "ProSafety Kazakhstan logo design.png"))

	// Home dashboard
	homeHandler := handlers.NewHomeHandler(itemsSvc, usersSvc, txSvc, deptsSvc)
	r.GET("/", homeHandler.Dashboard)

	dashboardHandler := handlers.NewDashboardHandlerWithDepartments(itemsSvc, txSvc, usersSvc, deptsSvc)
	r.GET("/dashboard", dashboardHandler.Dashboard)

	// Items
	itemsHandler := handlers.NewItemsHandler(itemsSvc, txSvc, templates)
	r.GET("/items", itemsHandler.ListPage)
	r.POST("/items", itemsHandler.Add)
	r.GET("/items/export", itemsHandler.Export)

	// Users
	usersHandler := handlers.NewUsersHandler(usersSvc, templates)
	r.GET("/employees", usersHandler.ListPage)
	r.POST("/employees", usersHandler.Add)
	r.GET("/employees/export", usersHandler.Export)
	r.GET("/users", usersHandler.ListPage)
	r.POST("/users", usersHandler.Add)
	r.GET("/users/export", usersHandler.Export)

	// Transactions
	txHandler := handlers.NewTransactionsHandler(itemsSvc, txSvc, usersSvc, deptsSvc, templates)
	r.GET("/issue", txHandler.IssuePage)
	r.GET("/return", txHandler.ReturnPage)
	r.GET("/transactions", txHandler.History)
	r.POST("/issue", txHandler.Issue)
	r.POST("/return", txHandler.Return)
	r.GET("/transactions/export", txHandler.Export)

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
		"templates/main.html",
		"templates/dashboard.html",
		"templates/issue.html",
		"templates/return.html",
		"templates/employees.html",
		"templates/items.html",
		"templates/users.html",
		"templates/transactions.html",
		"templates/partials/items_table.html",
		"templates/partials/users_table.html",
		"templates/partials/transactions_table.html",
	}

	// Parse all templates into the same template set so `{{template "..."}}` works.
	template.Must(t.ParseFiles(paths...))
	return t
}

