package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupComplaintRoutes настраивает маршруты для управления жалобами
func SetupComplaintRoutes(app *fiber.App, complaintController *controllers.ComplaintController) {
	// Группа маршрутов для жалоб
	complaints := app.Group("/events")

	// POST /events/:id/complaints - подать жалобу на участника (требует авторизации)
	complaints.Post("/:id/complaints", complaintController.SubmitComplaint)

	// Группа маршрутов для управления жалобами
	complaintManagement := app.Group("/complaints")

	// GET /complaints - получить список жалоб (только для модераторов, требует авторизации)
	complaintManagement.Get("/", complaintController.GetComplaints)

	// PUT /complaints/:id/status - обновить статус жалобы (только для модераторов, требует авторизации)
	complaintManagement.Put("/:id/status", complaintController.UpdateComplaintStatus)

	// GET /complaints/reasons - получить список причин жалоб (публичный доступ)
	complaintManagement.Get("/reasons", complaintController.GetComplaintReasons)

	// GET /complaints/my - получить жалобы пользователя (требует авторизации)
	complaintManagement.Get("/my", complaintController.GetUserComplaints)
}
