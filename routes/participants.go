package routes

import (
	"toloko-backend/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupParticipantRoutes настраивает маршруты для управления участниками ивентов
func SetupParticipantRoutes(app *fiber.App, participantController *controllers.ParticipantController) {
	// Группа маршрутов для участников ивентов
	participants := app.Group("/events")

	// POST /events/:id/join - вступить в ивент или подать заявку (требует авторизации)
	participants.Post("/:id/join", participantController.JoinEvent)

	// POST /events/:id/leave - выйти из ивента (требует авторизации)
	participants.Post("/:id/leave", participantController.LeaveEvent)

	// GET /events/:id/participants - получить список участников (требует авторизации)
	participants.Get("/:id/participants", participantController.GetParticipants)

	// GET /events/:id/applications - получить список заявок (только для создателя, требует авторизации)
	participants.Get("/:id/applications", participantController.GetApplications)

	// POST /events/:id/applications/:application_id/approve - одобрить заявку (только для создателя, требует авторизации)
	participants.Post("/:id/applications/:application_id/approve", participantController.ApproveApplication)

	// POST /events/:id/applications/:application_id/reject - отклонить заявку (только для создателя, требует авторизации)
	participants.Post("/:id/applications/:application_id/reject", participantController.RejectApplication)

	// POST /events/:id/complete - завершить ивент (только для создателя, требует авторизации)
	participants.Post("/:id/complete", participantController.CompleteEvent)

	// GET /events/:id/inventory-summary - получить сводку по инвентарю (только для создателя, требует авторизации)
	participants.Get("/:id/inventory-summary", participantController.GetInventorySummary)

	// Группа маршрутов для инвентаря участников
	participantInventory := app.Group("/participants")

	// POST /participants/:participant_id/inventory - обновить инвентарь участника (требует авторизации)
	participantInventory.Post("/:participant_id/inventory", participantController.UpdateParticipantInventory)
}
