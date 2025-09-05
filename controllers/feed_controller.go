package controllers

import (
	"strconv"
	"time"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// FeedController контроллер для управления лентой событий
type FeedController struct {
	DB *gorm.DB
}

// NewFeedController создает новый экземпляр FeedController
func NewFeedController(db *gorm.DB) *FeedController {
	return &FeedController{DB: db}
}

// FeedResponse структура ответа ленты
type FeedResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// GetFeed возвращает ленту событий пользователей, на которых подписан текущий пользователь
func (fc *FeedController) GetFeed(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := fc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(FeedResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем параметры пагинации
	page, limit := fc.getPaginationParams(c)

	// Получаем ID пользователей, на которых подписан текущий пользователь
	var subscriptionIDs []uint
	err = fc.DB.Model(&models.Subscription{}).
		Where("subscriber_id = ?", userID).
		Pluck("subscribed_to_id", &subscriptionIDs).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении подписок",
		})
	}

	// Если пользователь ни на кого не подписан, возвращаем пустую ленту
	if len(subscriptionIDs) == 0 {
		return c.JSON(FeedResponse{
			Success: true,
			Message: "Лента пуста - вы ни на кого не подписаны",
			Data: fiber.Map{
				"events": []models.Event{},
				"total":  0,
				"page":   page,
				"limit":  limit,
			},
		})
	}

	// Получаем события от подписанных пользователей
	var events []models.Event
	var total int64

	// Подсчитываем общее количество событий
	fc.DB.Model(&models.Event{}).
		Where("creator_id IN ? AND is_active = ?", subscriptionIDs, true).
		Count(&total)

	// Получаем события с пагинацией
	offset := (page - 1) * limit
	err = fc.DB.Preload("Creator").
		Preload("Photos").
		Preload("Inventory.Inventory").
		Preload("Participants.User").
		Where("creator_id IN ? AND is_active = ?", subscriptionIDs, true).
		Order("start_time ASC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении ленты",
		})
	}

	// Фильтруем события по времени (показываем только будущие и недавно прошедшие)
	now := time.Now()
	var filteredEvents []models.Event
	for _, event := range events {
		// Показываем события, которые начались не более чем за 7 дней до текущего времени
		// и еще не закончились более чем 1 день назад
		if event.StartTime.After(now.AddDate(0, 0, -7)) && event.EndTime.After(now.AddDate(0, 0, -1)) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return c.JSON(FeedResponse{
		Success: true,
		Message: "Лента получена успешно",
		Data: fiber.Map{
			"events": filteredEvents,
			"total":  len(filteredEvents),
			"page":   page,
			"limit":  limit,
		},
	})
}

// GetFeedWithFilters возвращает ленту с дополнительными фильтрами
func (fc *FeedController) GetFeedWithFilters(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := fc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(FeedResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем параметры пагинации
	page, limit := fc.getPaginationParams(c)

	// Получаем фильтры из query параметров
	status := c.Query("status", "all") // all, upcoming, past, today
	search := c.Query("search", "")

	// Получаем ID пользователей, на которых подписан текущий пользователь
	var subscriptionIDs []uint
	err = fc.DB.Model(&models.Subscription{}).
		Where("subscriber_id = ?", userID).
		Pluck("subscribed_to_id", &subscriptionIDs).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении подписок",
		})
	}

	// Если пользователь ни на кого не подписан, возвращаем пустую ленту
	if len(subscriptionIDs) == 0 {
		return c.JSON(FeedResponse{
			Success: true,
			Message: "Лента пуста - вы ни на кого не подписаны",
			Data: fiber.Map{
				"events": []models.Event{},
				"total":  0,
				"page":   page,
				"limit":  limit,
			},
		})
	}

	// Строим запрос с фильтрами
	query := fc.DB.Model(&models.Event{}).
		Where("creator_id IN ? AND is_active = ?", subscriptionIDs, true)

	// Применяем фильтр по статусу
	now := time.Now()
	switch status {
	case "upcoming":
		query = query.Where("start_time > ?", now)
	case "past":
		query = query.Where("end_time < ?", now)
	case "today":
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)
		query = query.Where("start_time >= ? AND start_time < ?", startOfDay, endOfDay)
	default: // "all"
		// Показываем события за последние 30 дней и будущие
		query = query.Where("start_time > ?", now.AddDate(0, 0, -30))
	}

	// Применяем поиск по названию и описанию
	if search != "" {
		query = query.Where("(title ILIKE ? OR description ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}

	// Подсчитываем общее количество событий
	var total int64
	query.Count(&total)

	// Получаем события с пагинацией
	offset := (page - 1) * limit
	var events []models.Event
	err = query.Preload("Creator").
		Preload("Photos").
		Preload("Inventory.Inventory").
		Preload("Participants.User").
		Order("start_time ASC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении ленты",
		})
	}

	return c.JSON(FeedResponse{
		Success: true,
		Message: "Лента получена успешно",
		Data: fiber.Map{
			"events": events,
			"total":  total,
			"page":   page,
			"limit":  limit,
			"filters": fiber.Map{
				"status": status,
				"search": search,
			},
		},
	})
}

// GetRecommendedEvents возвращает рекомендуемые события для пользователя
func (fc *FeedController) GetRecommendedEvents(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := fc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(FeedResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем параметры пагинации
	page, limit := fc.getPaginationParams(c)

	// Получаем ID пользователей, на которых подписан текущий пользователь
	var subscriptionIDs []uint
	err = fc.DB.Model(&models.Subscription{}).
		Where("subscriber_id = ?", userID).
		Pluck("subscribed_to_id", &subscriptionIDs).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении подписок",
		})
	}

	// Получаем рекомендуемые события (события от пользователей, на которых НЕ подписан текущий пользователь)
	var events []models.Event
	var total int64

	// Исключаем события от самого пользователя и от тех, на кого он уже подписан
	excludeIDs := append(subscriptionIDs, userID)

	// Подсчитываем общее количество рекомендуемых событий
	fc.DB.Model(&models.Event{}).
		Where("creator_id NOT IN ? AND is_active = ? AND start_time > ?", excludeIDs, true, time.Now()).
		Count(&total)

	// Получаем рекомендуемые события с пагинацией
	offset := (page - 1) * limit
	err = fc.DB.Preload("Creator").
		Preload("Photos").
		Preload("Inventory.Inventory").
		Preload("Participants.User").
		Where("creator_id NOT IN ? AND is_active = ? AND start_time > ?", excludeIDs, true, time.Now()).
		Order("start_time ASC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении рекомендуемых событий",
		})
	}

	return c.JSON(FeedResponse{
		Success: true,
		Message: "Рекомендуемые события получены успешно",
		Data: fiber.Map{
			"events": events,
			"total":  total,
			"page":   page,
			"limit":  limit,
		},
	})
}

// GetFeedStats возвращает статистику ленты пользователя
func (fc *FeedController) GetFeedStats(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := fc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(FeedResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем ID пользователей, на которых подписан текущий пользователь
	var subscriptionIDs []uint
	err = fc.DB.Model(&models.Subscription{}).
		Where("subscriber_id = ?", userID).
		Pluck("subscribed_to_id", &subscriptionIDs).Error

	if err != nil {
		return c.Status(500).JSON(FeedResponse{
			Success: false,
			Message: "Ошибка при получении подписок",
		})
	}

	now := time.Now()
	stats := fiber.Map{
		"subscriptions_count": len(subscriptionIDs),
		"upcoming_events":     0,
		"past_events":         0,
		"today_events":        0,
	}

	if len(subscriptionIDs) > 0 {
		// Подсчитываем предстоящие события
		var upcomingCount int64
		fc.DB.Model(&models.Event{}).
			Where("creator_id IN ? AND is_active = ? AND start_time > ?", subscriptionIDs, true, now).
			Count(&upcomingCount)
		stats["upcoming_events"] = upcomingCount

		// Подсчитываем прошедшие события
		var pastCount int64
		fc.DB.Model(&models.Event{}).
			Where("creator_id IN ? AND is_active = ? AND end_time < ?", subscriptionIDs, true, now).
			Count(&pastCount)
		stats["past_events"] = pastCount

		// Подсчитываем события на сегодня
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)
		var todayCount int64
		fc.DB.Model(&models.Event{}).
			Where("creator_id IN ? AND is_active = ? AND start_time >= ? AND start_time < ?", subscriptionIDs, true, startOfDay, endOfDay).
			Count(&todayCount)
		stats["today_events"] = todayCount
	}

	return c.JSON(FeedResponse{
		Success: true,
		Message: "Статистика ленты получена успешно",
		Data:    stats,
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (fc *FeedController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0, fiber.NewError(401, "Отсутствует токен авторизации")
	}

	// Извлекаем токен из заголовка "Bearer <token>"
	token := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	// Валидируем токен
	claims, err := utils.ValidateJWT(token)
	if err != nil {
		return 0, fiber.NewError(401, "Недействительный токен")
	}

	return claims.UserID, nil
}

// getPaginationParams извлекает параметры пагинации из запроса
func (fc *FeedController) getPaginationParams(c *fiber.Ctx) (int, int) {
	page := 1
	limit := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
}
