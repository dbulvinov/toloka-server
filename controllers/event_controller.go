package controllers

import (
	"strconv"
	"strings"
	"time"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// EventController контроллер для управления ивентами
type EventController struct {
	DB *gorm.DB
}

// NewEventController создает новый экземпляр EventController
func NewEventController(db *gorm.DB) *EventController {
	return &EventController{DB: db}
}

// CreateEventRequest структура запроса создания ивента
type CreateEventRequest struct {
	Title           string                  `json:"title" validate:"required,min=3,max=255"`
	Description     string                  `json:"description" validate:"max=2000"`
	Latitude        float64                 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude       float64                 `json:"longitude" validate:"required,min=-180,max=180"`
	StartTime       string                  `json:"start_time" validate:"required"`
	EndTime         string                  `json:"end_time" validate:"required"`
	JoinMode        string                  `json:"join_mode" validate:"required,oneof=free approval"`
	MinParticipants int                     `json:"min_participants" validate:"min=1"`
	MaxParticipants int                     `json:"max_participants" validate:"min=1"`
	Inventory       []EventInventoryRequest `json:"inventory"`
	Photos          []string                `json:"photos"`
}

// EventInventoryRequest структура запроса инвентаря для ивента
type EventInventoryRequest struct {
	InventoryID uint `json:"inventory_id"`
	QuantityMin int  `json:"quantity_min" validate:"min=1"`
	QuantityMax int  `json:"quantity_max" validate:"min=1"`
}

// UpdateEventRequest структура запроса обновления ивента
type UpdateEventRequest struct {
	Title           string                  `json:"title" validate:"min=3,max=255"`
	Description     string                  `json:"description" validate:"max=2000"`
	Latitude        float64                 `json:"latitude" validate:"min=-90,max=90"`
	Longitude       float64                 `json:"longitude" validate:"min=-180,max=180"`
	StartTime       string                  `json:"start_time"`
	EndTime         string                  `json:"end_time"`
	JoinMode        string                  `json:"join_mode" validate:"oneof=free approval"`
	MinParticipants int                     `json:"min_participants" validate:"min=1"`
	MaxParticipants int                     `json:"max_participants" validate:"min=1"`
	Inventory       []EventInventoryRequest `json:"inventory"`
	Photos          []string                `json:"photos"`
}

// CreateInventoryRequest структура запроса создания инвентаря
type CreateInventoryRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

// EventResponse структура ответа с ивентом
type EventResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Event   *models.Event `json:"event,omitempty"`
}

// EventsResponse структура ответа со списком ивентов
type EventsResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Events  []models.Event `json:"events,omitempty"`
	Total   int64          `json:"total,omitempty"`
}

// InventoryResponse структура ответа с инвентарем
type InventoryResponse struct {
	Success   bool              `json:"success"`
	Message   string            `json:"message"`
	Inventory *models.Inventory `json:"inventory,omitempty"`
}

// InventoriesResponse структура ответа со списком инвентаря
type InventoriesResponse struct {
	Success     bool               `json:"success"`
	Message     string             `json:"message"`
	Inventories []models.Inventory `json:"inventories,omitempty"`
}

// CreateEvent создает новый ивент
func (ec *EventController) CreateEvent(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := ec.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(EventResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	var req CreateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := ec.validateCreateEventRequest(&req); err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Парсим время
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный формат времени начала",
		})
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный формат времени окончания",
		})
	}

	// Проверяем, что время окончания после времени начала
	if endTime.Before(startTime) {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Время окончания должно быть после времени начала",
		})
	}

	// Проверяем, что время начала в будущем
	if startTime.Before(time.Now()) {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Время начала должно быть в будущем",
		})
	}

	// Проверяем количество участников
	if req.MaxParticipants < req.MinParticipants {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Максимальное количество участников должно быть больше или равно минимальному",
		})
	}

	// Создаем ивент
	event := models.Event{
		CreatorID:       userID,
		Title:           req.Title,
		Description:     req.Description,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		StartTime:       startTime,
		EndTime:         endTime,
		JoinMode:        req.JoinMode,
		MinParticipants: req.MinParticipants,
		MaxParticipants: req.MaxParticipants,
		IsActive:        true,
	}

	// Начинаем транзакцию
	tx := ec.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Создаем ивент
	if err := tx.Create(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при создании ивента",
		})
	}

	// Добавляем инвентарь
	for _, inv := range req.Inventory {
		// Проверяем существование инвентаря
		var inventory models.Inventory
		if err := tx.First(&inventory, inv.InventoryID).Error; err != nil {
			tx.Rollback()
			return c.Status(400).JSON(EventResponse{
				Success: false,
				Message: "Инвентарь с ID " + strconv.Itoa(int(inv.InventoryID)) + " не найден",
			})
		}

		eventInventory := models.EventInventory{
			EventID:     event.ID,
			InventoryID: inv.InventoryID,
			QuantityMin: inv.QuantityMin,
			QuantityMax: inv.QuantityMax,
		}

		if err := tx.Create(&eventInventory).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(EventResponse{
				Success: false,
				Message: "Ошибка при добавлении инвентаря",
			})
		}
	}

	// Добавляем фотографии
	for _, photoPath := range req.Photos {
		eventPhoto := models.EventPhoto{
			EventID:  event.ID,
			FilePath: photoPath,
			IsBefore: true, // По умолчанию фото до уборки
		}

		if err := tx.Create(&eventPhoto).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(EventResponse{
				Success: false,
				Message: "Ошибка при добавлении фотографии",
			})
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при сохранении ивента",
		})
	}

	// Загружаем полную информацию об ивенте
	if err := ec.DB.Preload("Creator").Preload("Inventory.Inventory").Preload("Photos").First(&event, event.ID).Error; err != nil {
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при загрузке данных ивента",
		})
	}

	return c.Status(201).JSON(EventResponse{
		Success: true,
		Message: "Ивент успешно создан",
		Event:   &event,
	})
}

// UpdateEvent обновляет существующий ивент
func (ec *EventController) UpdateEvent(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := ec.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(EventResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента и права доступа
	var event models.Event
	if err := ec.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(EventResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	if event.CreatorID != userID {
		return c.Status(403).JSON(EventResponse{
			Success: false,
			Message: "Нет прав для редактирования этого ивента",
		})
	}

	var req UpdateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := ec.validateUpdateEventRequest(&req); err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Обновляем поля
	if req.Title != "" {
		event.Title = req.Title
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if req.Latitude != 0 {
		event.Latitude = req.Latitude
	}
	if req.Longitude != 0 {
		event.Longitude = req.Longitude
	}
	if req.JoinMode != "" {
		event.JoinMode = req.JoinMode
	}
	if req.MinParticipants > 0 {
		event.MinParticipants = req.MinParticipants
	}
	if req.MaxParticipants > 0 {
		event.MaxParticipants = req.MaxParticipants
	}

	// Обновляем время, если предоставлено
	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return c.Status(400).JSON(EventResponse{
				Success: false,
				Message: "Неверный формат времени начала",
			})
		}
		event.StartTime = startTime
	}

	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return c.Status(400).JSON(EventResponse{
				Success: false,
				Message: "Неверный формат времени окончания",
			})
		}
		event.EndTime = endTime
	}

	// Проверяем валидность времени
	if event.EndTime.Before(event.StartTime) {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Время окончания должно быть после времени начала",
		})
	}

	// Проверяем количество участников
	if event.MaxParticipants < event.MinParticipants {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Максимальное количество участников должно быть больше или равно минимальному",
		})
	}

	// Начинаем транзакцию
	tx := ec.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Обновляем ивент
	if err := tx.Save(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при обновлении ивента",
		})
	}

	// Обновляем инвентарь, если предоставлен
	if req.Inventory != nil {
		// Удаляем старый инвентарь
		if err := tx.Where("event_id = ?", event.ID).Delete(&models.EventInventory{}).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(EventResponse{
				Success: false,
				Message: "Ошибка при обновлении инвентаря",
			})
		}

		// Добавляем новый инвентарь
		for _, inv := range req.Inventory {
			// Проверяем существование инвентаря
			var inventory models.Inventory
			if err := tx.First(&inventory, inv.InventoryID).Error; err != nil {
				tx.Rollback()
				return c.Status(400).JSON(EventResponse{
					Success: false,
					Message: "Инвентарь с ID " + strconv.Itoa(int(inv.InventoryID)) + " не найден",
				})
			}

			eventInventory := models.EventInventory{
				EventID:     event.ID,
				InventoryID: inv.InventoryID,
				QuantityMin: inv.QuantityMin,
				QuantityMax: inv.QuantityMax,
			}

			if err := tx.Create(&eventInventory).Error; err != nil {
				tx.Rollback()
				return c.Status(500).JSON(EventResponse{
					Success: false,
					Message: "Ошибка при добавлении инвентаря",
				})
			}
		}
	}

	// Обновляем фотографии, если предоставлены
	if req.Photos != nil {
		// Удаляем старые фотографии
		if err := tx.Where("event_id = ?", event.ID).Delete(&models.EventPhoto{}).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(EventResponse{
				Success: false,
				Message: "Ошибка при обновлении фотографий",
			})
		}

		// Добавляем новые фотографии
		for _, photoPath := range req.Photos {
			eventPhoto := models.EventPhoto{
				EventID:  event.ID,
				FilePath: photoPath,
				IsBefore: true,
			}

			if err := tx.Create(&eventPhoto).Error; err != nil {
				tx.Rollback()
				return c.Status(500).JSON(EventResponse{
					Success: false,
					Message: "Ошибка при добавлении фотографии",
				})
			}
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при сохранении изменений",
		})
	}

	// Загружаем полную информацию об ивенте
	if err := ec.DB.Preload("Creator").Preload("Inventory.Inventory").Preload("Photos").First(&event, event.ID).Error; err != nil {
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при загрузке данных ивента",
		})
	}

	return c.JSON(EventResponse{
		Success: true,
		Message: "Ивент успешно обновлен",
		Event:   &event,
	})
}

// DeleteEvent удаляет ивент
func (ec *EventController) DeleteEvent(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := ec.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(EventResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента и права доступа
	var event models.Event
	if err := ec.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(EventResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	if event.CreatorID != userID {
		return c.Status(403).JSON(EventResponse{
			Success: false,
			Message: "Нет прав для удаления этого ивента",
		})
	}

	// Начинаем транзакцию
	tx := ec.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Удаляем связанные данные
	if err := tx.Where("event_id = ?", event.ID).Delete(&models.EventInventory{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при удалении инвентаря",
		})
	}

	if err := tx.Where("event_id = ?", event.ID).Delete(&models.EventPhoto{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при удалении фотографий",
		})
	}

	if err := tx.Where("event_id = ?", event.ID).Delete(&models.EventParticipant{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при удалении участников",
		})
	}

	// Удаляем ивент
	if err := tx.Delete(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при удалении ивента",
		})
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(EventResponse{
			Success: false,
			Message: "Ошибка при удалении ивента",
		})
	}

	return c.JSON(EventResponse{
		Success: true,
		Message: "Ивент успешно удален",
	})
}

// GetEvents получает список ивентов с фильтрацией
func (ec *EventController) GetEvents(c *fiber.Ctx) error {
	// Получаем параметры запроса
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	_ = c.Query("city")         // Пока не используется, но может быть полезно в будущем
	status := c.Query("status") // "active", "completed", "upcoming"
	search := c.Query("search")

	// Валидация параметров
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Строим запрос
	query := ec.DB.Model(&models.Event{}).Preload("Creator").Preload("Inventory.Inventory").Preload("Photos")

	// Фильтр по статусу
	now := time.Now()
	switch status {
	case "active":
		query = query.Where("start_time <= ? AND end_time >= ?", now, now)
	case "completed":
		query = query.Where("end_time < ?", now)
	case "upcoming":
		query = query.Where("start_time > ?", now)
	}

	// Фильтр по поиску
	if search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Фильтр по городу (пока не реализован, так как нет поля city в модели)
	// В будущем можно добавить поле city или использовать геолокацию

	// Получаем общее количество
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(EventsResponse{
			Success: false,
			Message: "Ошибка при получении списка ивентов",
		})
	}

	// Получаем ивенты
	var events []models.Event
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&events).Error; err != nil {
		return c.Status(500).JSON(EventsResponse{
			Success: false,
			Message: "Ошибка при получении списка ивентов",
		})
	}

	return c.JSON(EventsResponse{
		Success: true,
		Message: "Список ивентов получен",
		Events:  events,
		Total:   total,
	})
}

// GetEvent получает детали конкретного ивента
func (ec *EventController) GetEvent(c *fiber.Ctx) error {
	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(EventResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Получаем ивент
	var event models.Event
	if err := ec.DB.Preload("Creator").Preload("Inventory.Inventory").Preload("Photos").Preload("Participants.User").First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(EventResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	return c.JSON(EventResponse{
		Success: true,
		Message: "Ивент найден",
		Event:   &event,
	})
}

// CreateInventory создает новый инвентарь
func (ec *EventController) CreateInventory(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := ec.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(InventoryResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	var req CreateInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(InventoryResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if strings.TrimSpace(req.Name) == "" {
		return c.Status(400).JSON(InventoryResponse{
			Success: false,
			Message: "Название инвентаря обязательно",
		})
	}

	// Проверяем, не существует ли уже такой инвентарь
	var existingInventory models.Inventory
	if err := ec.DB.Where("name = ?", strings.TrimSpace(req.Name)).First(&existingInventory).Error; err == nil {
		return c.Status(409).JSON(InventoryResponse{
			Success: false,
			Message: "Инвентарь с таким названием уже существует",
		})
	}

	// Создаем инвентарь
	inventory := models.Inventory{
		Name:      strings.TrimSpace(req.Name),
		IsCustom:  true,
		CreatedBy: userID,
		IsActive:  true,
	}

	if err := ec.DB.Create(&inventory).Error; err != nil {
		return c.Status(500).JSON(InventoryResponse{
			Success: false,
			Message: "Ошибка при создании инвентаря",
		})
	}

	return c.Status(201).JSON(InventoryResponse{
		Success:   true,
		Message:   "Инвентарь успешно создан",
		Inventory: &inventory,
	})
}

// GetInventories получает список инвентаря
func (ec *EventController) GetInventories(c *fiber.Ctx) error {
	// Получаем параметры запроса
	search := c.Query("search")
	custom := c.Query("custom") // "true" или "false"

	// Строим запрос
	query := ec.DB.Model(&models.Inventory{}).Where("is_active = ?", true)

	// Фильтр по типу (системный/пользовательский)
	if custom == "true" {
		query = query.Where("is_custom = ?", true)
	} else if custom == "false" {
		query = query.Where("is_custom = ?", false)
	}

	// Фильтр по поиску
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	// Получаем инвентарь
	var inventories []models.Inventory
	if err := query.Order("name ASC").Find(&inventories).Error; err != nil {
		return c.Status(500).JSON(InventoriesResponse{
			Success: false,
			Message: "Ошибка при получении списка инвентаря",
		})
	}

	return c.JSON(InventoriesResponse{
		Success:     true,
		Message:     "Список инвентаря получен",
		Inventories: inventories,
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (ec *EventController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
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
	claims, err := utils.ValidateJWT(token)
	if err != nil {
		return 0, fiber.NewError(401, "Недействительный токен")
	}

	return claims.UserID, nil
}

// validateCreateEventRequest валидирует запрос создания ивента
func (ec *EventController) validateCreateEventRequest(req *CreateEventRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return fiber.NewError(400, "Название ивента обязательно")
	}
	if len(req.Title) < 3 || len(req.Title) > 255 {
		return fiber.NewError(400, "Название должно содержать от 3 до 255 символов")
	}
	if len(req.Description) > 2000 {
		return fiber.NewError(400, "Описание не должно превышать 2000 символов")
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		return fiber.NewError(400, "Неверная широта")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return fiber.NewError(400, "Неверная долгота")
	}
	if req.JoinMode != "free" && req.JoinMode != "approval" {
		return fiber.NewError(400, "Режим вступления должен быть 'free' или 'approval'")
	}
	if req.MinParticipants < 1 {
		return fiber.NewError(400, "Минимальное количество участников должно быть больше 0")
	}
	if req.MaxParticipants < 1 {
		return fiber.NewError(400, "Максимальное количество участников должно быть больше 0")
	}
	return nil
}

// validateUpdateEventRequest валидирует запрос обновления ивента
func (ec *EventController) validateUpdateEventRequest(req *UpdateEventRequest) error {
	if req.Title != "" && (len(req.Title) < 3 || len(req.Title) > 255) {
		return fiber.NewError(400, "Название должно содержать от 3 до 255 символов")
	}
	if req.Description != "" && len(req.Description) > 2000 {
		return fiber.NewError(400, "Описание не должно превышать 2000 символов")
	}
	if req.Latitude != 0 && (req.Latitude < -90 || req.Latitude > 90) {
		return fiber.NewError(400, "Неверная широта")
	}
	if req.Longitude != 0 && (req.Longitude < -180 || req.Longitude > 180) {
		return fiber.NewError(400, "Неверная долгота")
	}
	if req.JoinMode != "" && req.JoinMode != "free" && req.JoinMode != "approval" {
		return fiber.NewError(400, "Режим вступления должен быть 'free' или 'approval'")
	}
	if req.MinParticipants > 0 && req.MinParticipants < 1 {
		return fiber.NewError(400, "Минимальное количество участников должно быть больше 0")
	}
	if req.MaxParticipants > 0 && req.MaxParticipants < 1 {
		return fiber.NewError(400, "Максимальное количество участников должно быть больше 0")
	}
	return nil
}
