package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupFeedRoutes настраивает маршруты для ленты событий
func SetupFeedRoutes(app *fiber.App, feedController *controllers.FeedController) {
	// Группа маршрутов для ленты
	feed := app.Group("/feed")

	// GET /feed - основная лента событий от подписанных пользователей
	feed.Get("/", feedController.GetFeed)

	// GET /feed/filtered - лента с фильтрами
	feed.Get("/filtered", feedController.GetFeedWithFilters)

	// GET /feed/recommended - рекомендуемые события
	feed.Get("/recommended", feedController.GetRecommendedEvents)

	// GET /feed/stats - статистика ленты
	feed.Get("/stats", feedController.GetFeedStats)
}
