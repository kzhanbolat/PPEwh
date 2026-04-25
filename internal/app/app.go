package app

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	hasher, err := services.NewPasswordHasher(filepath.Join(dataDir, "password.key"))
	if err != nil {
		panic(err)
	}
	authSvc := services.NewAuthService(csvStore.AuthAccounts(), hasher)
	if err := authSvc.EnsureDefaultAdmin(); err != nil {
		panic(err)
	}
	sessionSvc := services.NewSessionService(24 * time.Hour)

	templates := loadTemplates()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.SetHTMLTemplate(templates)
	r.Static("/static", filepath.Join(rootDir, "static"))
	r.StaticFile("/logo.png", filepath.Join(rootDir, "ProSafety Kazakhstan logo design.png"))
	r.StaticFile("/favicon.ico", filepath.Join(rootDir, "ProSafety Kazakhstan logo design.png"))
	r.NoRoute(func(c *gin.Context) {
		lang := i18n.NormalizeLang(c.Query("lang"))
		t := func(key string) string { return i18n.T(lang, key) }
		c.HTML(http.StatusNotFound, "error.html", handlers.PageData{
			Lang:  lang,
			T:     t,
			Error: t("page_not_found"),
			Data: map[string]string{
				"Title": t("not_found"),
			},
		})
	})

	authHandler := handlers.NewAuthHandler(authSvc, sessionSvc, usersSvc, deptsSvc)

	// Public auth pages
	r.GET("/login", authHandler.LoginPage)
	r.POST("/login", authHandler.Login)
	r.GET("/register", authHandler.RegisterPage)
	r.POST("/register", authHandler.Register)

	// Basic health check
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	protected := r.Group("/")
	protected.Use(handlers.RequireAuth(authSvc, sessionSvc))
	protected.POST("/logout", authHandler.Logout)
	protected.GET("/change-password", authHandler.ChangePasswordPage)
	protected.POST("/change-password", authHandler.ChangePassword)

	admin := protected.Group("/admin")
	admin.Use(handlers.RequireAdmin())
	admin.GET("/access", authHandler.AdminAccessPage)
	admin.POST("/access/approval", authHandler.AdminSetApproval)
	admin.POST("/access/reset-password", authHandler.AdminResetPassword)

	// Home dashboard
	homeHandler := handlers.NewHomeHandler(itemsSvc, usersSvc, txSvc, deptsSvc)
	protected.GET("/", homeHandler.Dashboard)

	dashboardHandler := handlers.NewDashboardHandlerWithDepartments(itemsSvc, txSvc, usersSvc, deptsSvc)
	protected.GET("/dashboard", dashboardHandler.Dashboard)

	// Items
	itemsHandler := handlers.NewItemsHandler(itemsSvc, txSvc, templates)
	protected.GET("/items", itemsHandler.ListPage)
	protected.POST("/items", itemsHandler.Add)
	protected.POST("/items/upload", itemsHandler.UploadExcel)
	protected.GET("/items/export", itemsHandler.Export)

	// Users
	usersHandler := handlers.NewUsersHandler(usersSvc, templates)
	protected.GET("/employees", usersHandler.ListPage)
	protected.GET("/employees/export", usersHandler.Export)
	protected.GET("/users", usersHandler.ListPage)
	protected.GET("/users/export", usersHandler.Export)
	protectedAdminEmployees := protected.Group("/")
	protectedAdminEmployees.Use(handlers.RequireAdmin())
	protectedAdminEmployees.POST("/employees", usersHandler.Add)
	protectedAdminEmployees.POST("/employees/upload", usersHandler.UploadExcel)
	protectedAdminEmployees.POST("/users", usersHandler.Add)
	protectedAdminEmployees.POST("/users/upload", usersHandler.UploadExcel)

	// Transactions
	txHandler := handlers.NewTransactionsHandler(itemsSvc, txSvc, usersSvc, deptsSvc, authSvc, templates)
	protected.GET("/issue", txHandler.IssuePage)
	protected.GET("/return", txHandler.ReturnPage)
	protected.GET("/transactions", txHandler.History)
	protected.POST("/issue", txHandler.Issue)
	protected.POST("/return", txHandler.Return)
	protected.GET("/transactions/export", txHandler.Export)

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
		"templates/auth_login.html",
		"templates/auth_register.html",
		"templates/auth_change_password.html",
		"templates/admin_access.html",
		"templates/error.html",
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

