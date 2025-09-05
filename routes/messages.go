package routes

import (
	"toloko-backend/controllers"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupMessageRoutes настраивает маршруты для сообщений
func SetupMessageRoutes(app *fiber.App, db *gorm.DB) {
	messageController := controllers.NewMessageController(db)

	// Группа маршрутов для сообщений
	messages := app.Group("/api/messages", utils.AuthMiddleware)

	// GET /api/messages/search - поиск сообщений
	messages.Get("/search", messageController.SearchMessages)

	// GET /api/messages/unread - получить непрочитанные сообщения
	messages.Get("/unread", messageController.GetUnreadMessages)

	// GET /api/messages/stats - получить статистику сообщений
	messages.Get("/stats", messageController.GetMessageStats)

	// GET /api/messages/:id - получить сообщение по ID
	messages.Get("/:id", messageController.GetMessage)

	// PUT /api/messages/:id/read - пометить сообщение как прочитанное
	messages.Put("/:id/read", messageController.MarkAsRead)

	// DELETE /api/messages/:id - удалить сообщение
	messages.Delete("/:id", messageController.DeleteMessage)

	// Группа маршрутов для сообщений в диалогах
	conversationMessages := app.Group("/api/conversations/:conversation_id/messages", utils.AuthMiddleware)

	// GET /api/conversations/:conversation_id/messages - получить сообщения диалога
	conversationMessages.Get("/", messageController.GetMessages)

	// PUT /api/conversations/:conversation_id/messages/read - пометить все сообщения диалога как прочитанные
	conversationMessages.Put("/read", messageController.MarkConversationAsRead)
}
