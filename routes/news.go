package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupNewsRoutes настраивает маршруты для работы с новостями
func SetupNewsRoutes(app *fiber.App, newsController *controllers.NewsController) {
	// Группа маршрутов для новостей сообществ
	communities := app.Group("/communities")

	// Маршруты для новостей в сообществах
	communities.Post("/:id/news", newsController.CreateNews) // POST /communities/:id/news - создать новость
	communities.Get("/:id/news", newsController.GetNews)     // GET /communities/:id/news - получить новости сообщества

	// Группа маршрутов для отдельных новостей
	news := app.Group("/news")

	// Публичные маршруты
	news.Get("/:id", newsController.GetNewsByID)   // GET /news/:id - получить новость по ID
	news.Get("/:id/like", newsController.LikeNews) // GET /news/:id/like - лайк/анлайк новости

	// Защищенные маршруты
	news.Put("/:id", newsController.UpdateNews)    // PUT /news/:id - обновить новость
	news.Delete("/:id", newsController.DeleteNews) // DELETE /news/:id - удалить новость
}
