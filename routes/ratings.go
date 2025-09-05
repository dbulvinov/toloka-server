package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupRatingRoutes настраивает маршруты для управления рейтингами
func SetupRatingRoutes(app *fiber.App, ratingController *controllers.RatingController) {
	// Группа маршрутов для рейтингов
	ratings := app.Group("/events")

	// POST /events/:id/photos - загрузить фотографии после завершения ивента (требует авторизации)
	ratings.Post("/:id/photos", ratingController.UploadPhotos)

	// POST /events/:id/ratings - отправить рейтинги участников (требует авторизации)
	ratings.Post("/:id/ratings", ratingController.SubmitRatings)

	// GET /events/:id/ratings - получить рейтинги по ивенту (требует авторизации)
	ratings.Get("/:id/ratings", ratingController.GetEventRatings)

	// Группа маршрутов для рейтингов пользователей
	userRatings := app.Group("/users")

	// GET /users/:id/ratings - получить рейтинг пользователя (публичный доступ)
	userRatings.Get("/:id/ratings", ratingController.GetUserRatings)
}
