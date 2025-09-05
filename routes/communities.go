package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupCommunityRoutes настраивает маршруты для работы с сообществами
func SetupCommunityRoutes(app *fiber.App, communityController *controllers.CommunityController) {
	// Группа маршрутов для сообществ
	communities := app.Group("/communities")

	// Публичные маршруты (не требуют авторизации)
	communities.Get("/", communityController.GetCommunities)  // GET /communities - список сообществ
	communities.Get("/:id", communityController.GetCommunity) // GET /communities/:id - получить сообщество

	// Защищенные маршруты (требуют авторизации)
	communities.Post("/", communityController.CreateCommunity)      // POST /communities - создать сообщество
	communities.Put("/:id", communityController.UpdateCommunity)    // PUT /communities/:id - обновить сообщество
	communities.Delete("/:id", communityController.DeleteCommunity) // DELETE /communities/:id - удалить сообщество
}
