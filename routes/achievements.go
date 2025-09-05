package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupAchievementRoutes настраивает маршруты для управления достижениями
func SetupAchievementRoutes(app *fiber.App, achievementController *controllers.AchievementController) {
	api := app.Group("/api")

	// Маршруты для достижений
	achievements := api.Group("/achievements")
	achievements.Get("/", achievementController.GetAllAchievements)                                   // GET /api/achievements - получить все достижения
	achievements.Get("/user/:id", achievementController.GetUserAchievements)                          // GET /api/achievements/user/:id - получить достижения пользователя
	achievements.Post("/user/:user_id/award/:achievement_id", achievementController.AwardAchievement) // POST /api/achievements/user/:user_id/award/:achievement_id - наградить достижением
}
