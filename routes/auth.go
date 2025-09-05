package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes настраивает маршруты для аутентификации
func SetupAuthRoutes(app *fiber.App, authController *controllers.AuthController) {
	// Группа маршрутов для аутентификации
	auth := app.Group("/auth")

	// POST /auth/register - регистрация пользователя
	auth.Post("/register", authController.Register)

	// POST /auth/login - вход пользователя
	auth.Post("/login", authController.Login)

	// POST /auth/recover - запрос на восстановление пароля
	auth.Post("/recover", authController.Recover)

	// POST /auth/oauth/google - авторизация через Google
	auth.Post("/oauth/google", authController.OAuth)

	// POST /auth/oauth/facebook - авторизация через Facebook
	auth.Post("/oauth/facebook", authController.OAuth)

	// GET /auth/health - проверка работоспособности
	auth.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Auth service is running",
			"timestamp": fiber.Map{
				"unix": fiber.Map{
					"seconds": c.Context().Time().Unix(),
				},
			},
		})
	})
}
