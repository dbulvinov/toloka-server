package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupEventRoutes настраивает маршруты для управления ивентами
func SetupEventRoutes(app *fiber.App, eventController *controllers.EventController) {
	// Группа маршрутов для ивентов
	events := app.Group("/events")

	// POST /events - создать ивент (требует авторизации)
	events.Post("/", eventController.CreateEvent)

	// PUT /events/:id - редактировать ивент (требует авторизации)
	events.Put("/:id", eventController.UpdateEvent)

	// DELETE /events/:id - удалить ивент (требует авторизации)
	events.Delete("/:id", eventController.DeleteEvent)

	// GET /events - список ивентов (публичный доступ)
	events.Get("/", eventController.GetEvents)

	// GET /events/health - проверка работоспособности (должен быть перед параметрическим маршрутом)
	events.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Events service is running",
			"timestamp": fiber.Map{
				"unix": fiber.Map{
					"seconds": c.Context().Time().Unix(),
				},
			},
		})
	})

	// GET /events/:id - получить детали ивента (публичный доступ)
	events.Get("/:id", eventController.GetEvent)

	// Группа маршрутов для инвентаря
	inventory := app.Group("/inventory")

	// GET /inventory/health - проверка работоспособности (должен быть перед параметрическим маршрутом)
	inventory.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Inventory service is running",
			"timestamp": fiber.Map{
				"unix": fiber.Map{
					"seconds": c.Context().Time().Unix(),
				},
			},
		})
	})

	// POST /inventory - создать инвентарь (требует авторизации)
	inventory.Post("/", eventController.CreateInventory)

	// GET /inventory - список инвентаря (публичный доступ)
	inventory.Get("/", eventController.GetInventories)
}
