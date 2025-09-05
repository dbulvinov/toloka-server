package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupCommentRoutes настраивает маршруты для работы с комментариями
func SetupCommentRoutes(app *fiber.App, commentController *controllers.CommentController) {
	// Группа маршрутов для комментариев к новостям
	news := app.Group("/news")

	// Маршруты для комментариев
	news.Post("/:id/comments", commentController.CreateComment) // POST /news/:id/comments - создать комментарий
	news.Get("/:id/comments", commentController.GetComments)    // GET /news/:id/comments - получить комментарии

	// Группа маршрутов для отдельных комментариев
	comments := app.Group("/comments")

	// Публичные маршруты
	comments.Get("/:id", commentController.GetComment) // GET /comments/:id - получить комментарий по ID

	// Защищенные маршруты
	comments.Put("/:id", commentController.UpdateComment)    // PUT /comments/:id - обновить комментарий
	comments.Delete("/:id", commentController.DeleteComment) // DELETE /comments/:id - удалить комментарий
}
