package routes

import (
	"toloko-backend/controllers"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupConversationRoutes настраивает маршруты для диалогов
func SetupConversationRoutes(app *fiber.App, db *gorm.DB) {
	conversationController := controllers.NewConversationController(db)

	// Группа маршрутов для диалогов
	conversations := app.Group("/api/conversations", utils.AuthMiddleware)

	// GET /api/conversations - получить список диалогов
	conversations.Get("/", conversationController.GetConversations)

	// GET /api/conversations/:id - получить диалог по ID
	conversations.Get("/:id", conversationController.GetConversation)

	// POST /api/conversations - создать новый диалог
	conversations.Post("/", conversationController.CreateConversation)

	// PUT /api/conversations/:id/read - пометить диалог как прочитанный
	conversations.Put("/:id/read", conversationController.MarkAsRead)

	// DELETE /api/conversations/:id - удалить диалог
	conversations.Delete("/:id", conversationController.DeleteConversation)

	// GET /api/conversations/:id/stats - получить статистику диалога
	conversations.Get("/:id/stats", conversationController.GetConversationStats)
}
