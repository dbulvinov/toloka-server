package controllers

import (
	"strconv"
	"strings"
	"time"

	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// ParticipantController контроллер для управления участниками ивентов
type ParticipantController struct {
	DB *gorm.DB
}

// NewParticipantController создает новый экземпляр ParticipantController
func NewParticipantController(db *gorm.DB) *ParticipantController {
	return &ParticipantController{DB: db}
}

// JoinEventRequest структура запроса вступления в ивент
type JoinEventRequest struct {
	DesiredInventory []ParticipantInventoryRequest `json:"desired_inventory"`
}

// ParticipantInventoryRequest структура запроса инвентаря участника
type ParticipantInventoryRequest struct {
	InventoryItemID *uint  `json:"inventory_item_id"` // nullable for custom items
	CustomName      string `json:"custom_name"`       // для пользовательских предметов
	Quantity        int    `json:"quantity" validate:"min=1"`
}

// UpdateInventoryRequest структура запроса обновления инвентаря участника
type UpdateInventoryRequest struct {
	Inventory []ParticipantInventoryRequest `json:"inventory"`
}

// CompleteEventRequest структура запроса завершения ивента
type CompleteEventRequest struct {
	// Дополнительные поля при необходимости
}

// ParticipantResponse структура ответа с участником
type ParticipantResponse struct {
	Success     bool                     `json:"success"`
	Message     string                   `json:"message"`
	Participant *models.EventParticipant `json:"participant,omitempty"`
}

// ParticipantsResponse структура ответа со списком участников
type ParticipantsResponse struct {
	Success      bool                      `json:"success"`
	Message      string                    `json:"message"`
	Participants []models.EventParticipant `json:"participants,omitempty"`
	Total        int64                     `json:"total,omitempty"`
}

// InventorySummaryResponse структура ответа со сводкой инвентаря
type InventorySummaryResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Summary []InventorySummaryItem `json:"summary,omitempty"`
}

// InventorySummaryItem элемент сводки инвентаря
type InventorySummaryItem struct {
	InventoryItemID *uint                  `json:"inventory_item_id"`
	CustomName      string                 `json:"custom_name"`
	ItemName        string                 `json:"item_name"`
	TotalQuantity   int                    `json:"total_quantity"`
	Participants    []InventoryParticipant `json:"participants"`
}

// InventoryParticipant участник с инвентарем
type InventoryParticipant struct {
	UserID   uint   `json:"user_id"`
	UserName string `json:"user_name"`
	Quantity int    `json:"quantity"`
	Brought  bool   `json:"brought"`
}

// JoinEvent вступает в ивент или подает заявку
func (pc *ParticipantController) JoinEvent(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента
	var event models.Event
	if err := pc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(ParticipantResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	// Проверяем, что ивент активен
	if !event.IsActive {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Ивент неактивен",
		})
	}

	// Проверяем, что ивент еще не начался
	if time.Now().After(event.StartTime) {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Нельзя вступить в ивент после его начала",
		})
	}

	// Проверяем, не является ли пользователь создателем ивента
	if event.CreatorID == userID {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Создатель ивента не может вступить в него как участник",
		})
	}

	// Проверяем, не участвует ли уже пользователь
	var existingParticipant models.EventParticipant
	if err := pc.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existingParticipant).Error; err == nil {
		return c.Status(409).JSON(ParticipantResponse{
			Success: false,
			Message: "Вы уже участвуете в этом ивенте",
		})
	}

	var req JoinEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация инвентаря
	if err := pc.validateInventoryRequest(req.DesiredInventory); err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Определяем статус в зависимости от режима ивента
	status := models.ParticipantStatusPending
	if event.JoinMode == "free" {
		status = models.ParticipantStatusJoined
	}

	// Начинаем транзакцию
	tx := pc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Создаем участника
	participant := models.EventParticipant{
		EventID: uint(eventID),
		UserID:  userID,
		Status:  status,
	}

	if status == models.ParticipantStatusJoined {
		now := time.Now()
		participant.JoinedAt = &now
	}

	if err := tx.Create(&participant).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при вступлении в ивент",
		})
	}

	// Добавляем инвентарь участника
	for _, inv := range req.DesiredInventory {
		participantInventory := models.ParticipantInventory{
			ParticipantID:   participant.ID,
			InventoryItemID: inv.InventoryItemID,
			CustomName:      inv.CustomName,
			Quantity:        inv.Quantity,
		}

		if err := tx.Create(&participantInventory).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(ParticipantResponse{
				Success: false,
				Message: "Ошибка при добавлении инвентаря",
			})
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при сохранении данных",
		})
	}

	// Загружаем полную информацию об участнике
	if err := pc.DB.Preload("User").Preload("Event").First(&participant, participant.ID).Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при загрузке данных участника",
		})
	}

	message := "Заявка подана успешно"
	if status == models.ParticipantStatusJoined {
		message = "Вы успешно вступили в ивент"
	}

	return c.Status(201).JSON(ParticipantResponse{
		Success:     true,
		Message:     message,
		Participant: &participant,
	})
}

// LeaveEvent выходит из ивента
func (pc *ParticipantController) LeaveEvent(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Находим участника
	var participant models.EventParticipant
	if err := pc.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err != nil {
		return c.Status(404).JSON(ParticipantResponse{
			Success: false,
			Message: "Вы не участвуете в этом ивенте",
		})
	}

	// Проверяем, можно ли выйти
	if participant.Status == models.ParticipantStatusLeft {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Вы уже вышли из ивента",
		})
	}

	// Обновляем статус
	participant.Status = models.ParticipantStatusLeft
	now := time.Now()
	participant.LeftAt = &now

	if err := pc.DB.Save(&participant).Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при выходе из ивента",
		})
	}

	return c.JSON(ParticipantResponse{
		Success: true,
		Message: "Вы успешно вышли из ивента",
	})
}

// GetParticipants получает список участников ивента
func (pc *ParticipantController) GetParticipants(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantsResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента
	var event models.Event
	if err := pc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(ParticipantsResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	// Проверяем права доступа (только создатель может видеть заявки)
	canSeeApplications := event.CreatorID == userID

	// Строим запрос
	query := pc.DB.Model(&models.EventParticipant{}).Where("event_id = ?", eventID)

	// Если не создатель, показываем только подтвержденных участников
	if !canSeeApplications {
		query = query.Where("status IN ?", []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted})
	}

	// Получаем участников
	var participants []models.EventParticipant
	if err := query.Preload("User").Order("created_at ASC").Find(&participants).Error; err != nil {
		return c.Status(500).JSON(ParticipantsResponse{
			Success: false,
			Message: "Ошибка при получении списка участников",
		})
	}

	// Получаем общее количество
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(ParticipantsResponse{
			Success: false,
			Message: "Ошибка при получении количества участников",
		})
	}

	return c.JSON(ParticipantsResponse{
		Success:      true,
		Message:      "Список участников получен",
		Participants: participants,
		Total:        total,
	})
}

// GetApplications получает список заявок на участие (только для создателя)
func (pc *ParticipantController) GetApplications(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantsResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента и права доступа
	var event models.Event
	if err := pc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(ParticipantsResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	if event.CreatorID != userID {
		return c.Status(403).JSON(ParticipantsResponse{
			Success: false,
			Message: "Нет прав для просмотра заявок",
		})
	}

	// Получаем заявки
	var participants []models.EventParticipant
	if err := pc.DB.Where("event_id = ? AND status = ?", eventID, models.ParticipantStatusPending).
		Preload("User").Preload("ParticipantInventory.InventoryItem").
		Order("created_at ASC").Find(&participants).Error; err != nil {
		return c.Status(500).JSON(ParticipantsResponse{
			Success: false,
			Message: "Ошибка при получении списка заявок",
		})
	}

	// Получаем общее количество
	var total int64
	if err := pc.DB.Model(&models.EventParticipant{}).Where("event_id = ? AND status = ?", eventID, models.ParticipantStatusPending).Count(&total).Error; err != nil {
		return c.Status(500).JSON(ParticipantsResponse{
			Success: false,
			Message: "Ошибка при получении количества заявок",
		})
	}

	return c.JSON(ParticipantsResponse{
		Success:      true,
		Message:      "Список заявок получен",
		Participants: participants,
		Total:        total,
	})
}

// ApproveApplication одобряет заявку на участие
func (pc *ParticipantController) ApproveApplication(c *fiber.Ctx) error {
	return pc.updateApplicationStatus(c, models.ParticipantStatusAccepted, "Заявка одобрена")
}

// RejectApplication отклоняет заявку на участие
func (pc *ParticipantController) RejectApplication(c *fiber.Ctx) error {
	return pc.updateApplicationStatus(c, models.ParticipantStatusRejected, "Заявка отклонена")
}

// updateApplicationStatus обновляет статус заявки
func (pc *ParticipantController) updateApplicationStatus(c *fiber.Ctx, status, message string) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента и заявки
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	applicationID, err := strconv.ParseUint(c.Params("application_id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный ID заявки",
		})
	}

	// Проверяем существование ивента и права доступа
	var event models.Event
	if err := pc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(ParticipantResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	if event.CreatorID != userID {
		return c.Status(403).JSON(ParticipantResponse{
			Success: false,
			Message: "Нет прав для изменения статуса заявки",
		})
	}

	// Находим заявку
	var participant models.EventParticipant
	if err := pc.DB.Where("id = ? AND event_id = ? AND status = ?", applicationID, eventID, models.ParticipantStatusPending).First(&participant).Error; err != nil {
		return c.Status(404).JSON(ParticipantResponse{
			Success: false,
			Message: "Заявка не найдена",
		})
	}

	// Обновляем статус
	participant.Status = status
	if status == models.ParticipantStatusAccepted {
		now := time.Now()
		participant.JoinedAt = &now
	}

	if err := pc.DB.Save(&participant).Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при обновлении статуса заявки",
		})
	}

	// Загружаем полную информацию об участнике
	if err := pc.DB.Preload("User").Preload("Event").First(&participant, participant.ID).Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при загрузке данных участника",
		})
	}

	return c.JSON(ParticipantResponse{
		Success:     true,
		Message:     message,
		Participant: &participant,
	})
}

// UpdateParticipantInventory обновляет инвентарь участника
func (pc *ParticipantController) UpdateParticipantInventory(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID участника
	participantID, err := strconv.ParseUint(c.Params("participant_id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный ID участника",
		})
	}

	// Проверяем существование участника и права доступа
	var participant models.EventParticipant
	if err := pc.DB.Preload("Event").First(&participant, participantID).Error; err != nil {
		return c.Status(404).JSON(ParticipantResponse{
			Success: false,
			Message: "Участник не найден",
		})
	}

	// Проверяем права (участник или создатель ивента)
	if participant.UserID != userID && participant.Event.CreatorID != userID {
		return c.Status(403).JSON(ParticipantResponse{
			Success: false,
			Message: "Нет прав для изменения инвентаря",
		})
	}

	var req UpdateInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация инвентаря
	if err := pc.validateInventoryRequest(req.Inventory); err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Начинаем транзакцию
	tx := pc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Удаляем старый инвентарь
	if err := tx.Where("participant_id = ?", participantID).Delete(&models.ParticipantInventory{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при обновлении инвентаря",
		})
	}

	// Добавляем новый инвентарь
	for _, inv := range req.Inventory {
		participantInventory := models.ParticipantInventory{
			ParticipantID:   uint(participantID),
			InventoryItemID: inv.InventoryItemID,
			CustomName:      inv.CustomName,
			Quantity:        inv.Quantity,
		}

		if err := tx.Create(&participantInventory).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(ParticipantResponse{
				Success: false,
				Message: "Ошибка при добавлении инвентаря",
			})
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при сохранении инвентаря",
		})
	}

	return c.JSON(ParticipantResponse{
		Success: true,
		Message: "Инвентарь участника обновлен",
	})
}

// GetInventorySummary получает сводку по инвентарю ивента
func (pc *ParticipantController) GetInventorySummary(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(InventorySummaryResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(InventorySummaryResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента и права доступа
	var event models.Event
	if err := pc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(InventorySummaryResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	if event.CreatorID != userID {
		return c.Status(403).JSON(InventorySummaryResponse{
			Success: false,
			Message: "Нет прав для просмотра сводки инвентаря",
		})
	}

	// Получаем инвентарь участников
	var participantInventories []models.ParticipantInventory
	if err := pc.DB.Where("participant_id IN (SELECT id FROM event_participants WHERE event_id = ? AND status IN ?)",
		eventID, []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted}).
		Preload("Participant.User").
		Preload("InventoryItem").
		Find(&participantInventories).Error; err != nil {
		return c.Status(500).JSON(InventorySummaryResponse{
			Success: false,
			Message: "Ошибка при получении сводки инвентаря",
		})
	}

	// Группируем по предметам
	inventoryMap := make(map[string]*InventorySummaryItem)
	for _, pi := range participantInventories {
		key := ""
		itemName := ""

		if pi.InventoryItemID != nil {
			key = "item_" + strconv.Itoa(int(*pi.InventoryItemID))
			itemName = pi.InventoryItem.Name
		} else {
			key = "custom_" + pi.CustomName
			itemName = pi.CustomName
		}

		if _, exists := inventoryMap[key]; !exists {
			inventoryMap[key] = &InventorySummaryItem{
				InventoryItemID: pi.InventoryItemID,
				CustomName:      pi.CustomName,
				ItemName:        itemName,
				TotalQuantity:   0,
				Participants:    []InventoryParticipant{},
			}
		}

		item := inventoryMap[key]
		item.TotalQuantity += pi.Quantity
		item.Participants = append(item.Participants, InventoryParticipant{
			UserID:   pi.Participant.UserID,
			UserName: pi.Participant.User.Name,
			Quantity: pi.Quantity,
			Brought:  pi.Brought,
		})
	}

	// Преобразуем в слайс
	var summary []InventorySummaryItem
	for _, item := range inventoryMap {
		summary = append(summary, *item)
	}

	return c.JSON(InventorySummaryResponse{
		Success: true,
		Message: "Сводка инвентаря получена",
		Summary: summary,
	})
}

// CompleteEvent завершает ивент (только для создателя)
func (pc *ParticipantController) CompleteEvent(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := pc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ParticipantResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента и права доступа
	var event models.Event
	if err := pc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(ParticipantResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	if event.CreatorID != userID {
		return c.Status(403).JSON(ParticipantResponse{
			Success: false,
			Message: "Нет прав для завершения ивента",
		})
	}

	// Проверяем, что ивент еще не завершен
	if !event.IsActive {
		return c.Status(400).JSON(ParticipantResponse{
			Success: false,
			Message: "Ивент уже завершен",
		})
	}

	// Деактивируем ивент
	event.IsActive = false
	if err := pc.DB.Save(&event).Error; err != nil {
		return c.Status(500).JSON(ParticipantResponse{
			Success: false,
			Message: "Ошибка при завершении ивента",
		})
	}

	return c.JSON(ParticipantResponse{
		Success: true,
		Message: "Ивент успешно завершен",
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (pc *ParticipantController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0, fiber.NewError(401, "Отсутствует токен авторизации")
	}

	// Извлекаем токен из заголовка "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return 0, fiber.NewError(401, "Неверный формат токена")
	}

	token := tokenParts[1]

	// Валидируем токен
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("toloko-secret-key-change-in-production"), nil
	})

	if err != nil {
		return 0, fiber.NewError(401, "Недействительный токен")
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		if userID, ok := claims["user_id"].(float64); ok {
			return uint(userID), nil
		}
	}

	return 0, fiber.NewError(401, "Недействительный токен")
}

// validateInventoryRequest валидирует запрос инвентаря
func (pc *ParticipantController) validateInventoryRequest(inventory []ParticipantInventoryRequest) error {
	for _, inv := range inventory {
		if inv.Quantity < 1 {
			return fiber.NewError(400, "Количество должно быть больше 0")
		}

		// Если указан ID предмета, проверяем его существование
		if inv.InventoryItemID != nil {
			var inventoryItem models.Inventory
			if err := pc.DB.First(&inventoryItem, *inv.InventoryItemID).Error; err != nil {
				return fiber.NewError(400, "Инвентарь с ID "+strconv.Itoa(int(*inv.InventoryItemID))+" не найден")
			}
		} else {
			// Для пользовательских предметов проверяем название
			if strings.TrimSpace(inv.CustomName) == "" {
				return fiber.NewError(400, "Название пользовательского предмета обязательно")
			}
		}
	}
	return nil
}
