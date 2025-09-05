package routes

import (
	"toloko-backend/controllers"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupAttachmentRoutes настраивает маршруты для вложений
func SetupAttachmentRoutes(app *fiber.App, db *gorm.DB) {
	attachmentController := controllers.NewAttachmentController(db)

	// Группа маршрутов для вложений
	attachments := app.Group("/api/attachments", utils.AuthMiddleware)

	// GET /api/attachments/:id - получить информацию о вложении
	attachments.Get("/:id", attachmentController.GetAttachmentInfo)

	// GET /api/attachments/:id/file - получить файл вложения
	attachments.Get("/:id/file", attachmentController.GetAttachmentFile)

	// DELETE /api/attachments/:id - удалить вложение
	attachments.Delete("/:id", attachmentController.DeleteAttachment)

	// Группа маршрутов для вложений сообщений
	messageAttachments := app.Group("/api/messages/:message_id/attachments", utils.AuthMiddleware)

	// POST /api/messages/:message_id/attachments - загрузить вложение для сообщения
	messageAttachments.Post("/", attachmentController.UploadAttachment)

	// GET /api/messages/:message_id/attachments - получить вложения сообщения
	messageAttachments.Get("/", attachmentController.GetMessageAttachments)
}
