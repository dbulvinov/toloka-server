package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupUserRoutes настраивает маршруты для управления профилями пользователей
func SetupUserRoutes(app *fiber.App, userController *controllers.UserController) {
	api := app.Group("/api")

	// Маршруты для профилей пользователей
	users := api.Group("/users")
	users.Get("/:id", userController.GetProfile)                     // GET /api/users/:id - получить профиль пользователя
	users.Put("/:id", userController.UpdateProfile)                  // PUT /api/users/:id - обновить профиль пользователя
	users.Get("/:id/events", userController.GetUserEvents)           // GET /api/users/:id/events - получить события пользователя
	users.Get("/:id/communities", userController.GetUserCommunities) // GET /api/users/:id/communities - получить сообщества пользователя
}
