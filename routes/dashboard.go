package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupDashboardRoutes настраивает маршруты для дашборда
func SetupDashboardRoutes(app *fiber.App, dashboardController *controllers.DashboardController) {
	api := app.Group("/api/dashboard")

	// Получение полных данных дашборда
	api.Get("/", dashboardController.GetDashboardData)
	
	// Получение упрощенных данных дашборда для текущего пользователя
	api.Get("/user", dashboardController.GetUserDashboard)
}
