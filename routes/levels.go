package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupLevelRoutes настраивает маршруты для управления уровнями пользователей
func SetupLevelRoutes(app *fiber.App, levelController *controllers.LevelController) {
	api := app.Group("/api")

	// Маршруты для уровней
	levels := api.Group("/levels")
	levels.Get("/user/:id", levelController.GetUserLevel)          // GET /api/levels/user/:id - получить уровень пользователя
	levels.Get("/leaderboard", levelController.GetLeaderboard)     // GET /api/levels/leaderboard - получить таблицу лидеров
	levels.Post("/user/:id/add-points", levelController.AddPoints) // POST /api/levels/user/:id/add-points - добавить очки пользователю
}
