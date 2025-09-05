package routes

import (
	"toloko-backend/controllers"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupBlockRoutes настраивает маршруты для блокировок
func SetupBlockRoutes(app *fiber.App, db *gorm.DB) {
	blockController := controllers.NewBlockController(db)

	// Группа маршрутов для блокировок
	blocks := app.Group("/api/blocks", utils.AuthMiddleware)

	// GET /api/blocks - получить список заблокированных пользователей
	blocks.Get("/", blockController.GetBlockedUsers)

	// GET /api/blocks/stats - получить статистику блокировок
	blocks.Get("/stats", blockController.GetBlockStats)

	// GET /api/blocks/:user_id - проверить, заблокирован ли пользователь
	blocks.Get("/:user_id", blockController.IsBlocked)

	// POST /api/blocks/:user_id - заблокировать пользователя
	blocks.Post("/:user_id", blockController.BlockUser)

	// DELETE /api/blocks/:user_id - разблокировать пользователя
	blocks.Delete("/:user_id", blockController.UnblockUser)

	// GET /api/blocks/:user_id/info - получить информацию о блокировке
	blocks.Get("/:user_id/info", blockController.GetBlockInfo)
}
