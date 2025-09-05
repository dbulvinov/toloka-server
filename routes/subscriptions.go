package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupSubscriptionRoutes настраивает маршруты для подписок
func SetupSubscriptionRoutes(app *fiber.App, subscriptionController *controllers.SubscriptionController) {
	// Группа маршрутов для подписок
	subscriptions := app.Group("/subscriptions")

	// POST /subscriptions/:user_id - подписка на пользователя
	subscriptions.Post("/:user_id", subscriptionController.Subscribe)

	// DELETE /subscriptions/:user_id - отписка от пользователя
	subscriptions.Delete("/:user_id", subscriptionController.Unsubscribe)

	// GET /subscriptions - список подписок текущего пользователя
	subscriptions.Get("/", subscriptionController.GetSubscriptions)

	// GET /subscriptions/stats - статистика подписок
	subscriptions.Get("/stats", subscriptionController.GetSubscriptionStats)

	// GET /subscriptions/check/:user_id - проверка подписки на пользователя
	subscriptions.Get("/check/:user_id", subscriptionController.CheckSubscription)

	// Группа маршрутов для подписчиков
	subscribers := app.Group("/subscribers")

	// GET /subscribers - список подписчиков текущего пользователя
	subscribers.Get("/", subscriptionController.GetSubscribers)
}
