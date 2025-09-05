package controllers

import (
	"strconv"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SubscriptionController контроллер для управления подписками
type SubscriptionController struct {
	DB *gorm.DB
}

// NewSubscriptionController создает новый экземпляр SubscriptionController
func NewSubscriptionController(db *gorm.DB) *SubscriptionController {
	return &SubscriptionController{DB: db}
}

// SubscribeRequest структура запроса подписки
type SubscribeRequest struct {
	UserID uint `json:"user_id" validate:"required"`
}

// SubscriptionResponse структура ответа подписки
type SubscriptionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Subscribe обрабатывает подписку на пользователя
func (sc *SubscriptionController) Subscribe(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := sc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(SubscriptionResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем ID пользователя для подписки из параметров URL
	subscribedToIDStr := c.Params("user_id")
	subscribedToID, err := strconv.ParseUint(subscribedToIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(SubscriptionResponse{
			Success: false,
			Message: "Неверный ID пользователя",
		})
	}

	// Проверяем, что пользователь не подписывается сам на себя
	if userID == uint(subscribedToID) {
		return c.Status(400).JSON(SubscriptionResponse{
			Success: false,
			Message: "Нельзя подписаться на самого себя",
		})
	}

	// Проверяем, существует ли пользователь для подписки
	var targetUser models.User
	if err := sc.DB.First(&targetUser, subscribedToID).Error; err != nil {
		return c.Status(404).JSON(SubscriptionResponse{
			Success: false,
			Message: "Пользователь не найден",
		})
	}

	// Проверяем, не подписан ли уже пользователь
	var existingSubscription models.Subscription
	err = sc.DB.Where("subscriber_id = ? AND subscribed_to_id = ?", userID, subscribedToID).First(&existingSubscription).Error
	if err == nil {
		return c.Status(409).JSON(SubscriptionResponse{
			Success: false,
			Message: "Вы уже подписаны на этого пользователя",
		})
	}

	// Создаем подписку
	subscription := models.Subscription{
		SubscriberID:   userID,
		SubscribedToID: uint(subscribedToID),
	}

	if err := sc.DB.Create(&subscription).Error; err != nil {
		return c.Status(500).JSON(SubscriptionResponse{
			Success: false,
			Message: "Ошибка при создании подписки",
		})
	}

	// Загружаем данные подписки с информацией о пользователе
	var createdSubscription models.Subscription
	sc.DB.Preload("SubscribedTo").First(&createdSubscription, subscription.ID)

	return c.Status(201).JSON(SubscriptionResponse{
		Success: true,
		Message: "Успешно подписались на пользователя",
		Data:    createdSubscription,
	})
}

// Unsubscribe обрабатывает отписку от пользователя
func (sc *SubscriptionController) Unsubscribe(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := sc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(SubscriptionResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем ID пользователя для отписки из параметров URL
	subscribedToIDStr := c.Params("user_id")
	subscribedToID, err := strconv.ParseUint(subscribedToIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(SubscriptionResponse{
			Success: false,
			Message: "Неверный ID пользователя",
		})
	}

	// Ищем подписку для удаления
	var subscription models.Subscription
	err = sc.DB.Where("subscriber_id = ? AND subscribed_to_id = ?", userID, subscribedToID).First(&subscription).Error
	if err != nil {
		return c.Status(404).JSON(SubscriptionResponse{
			Success: false,
			Message: "Подписка не найдена",
		})
	}

	// Удаляем подписку
	if err := sc.DB.Delete(&subscription).Error; err != nil {
		return c.Status(500).JSON(SubscriptionResponse{
			Success: false,
			Message: "Ошибка при удалении подписки",
		})
	}

	return c.JSON(SubscriptionResponse{
		Success: true,
		Message: "Успешно отписались от пользователя",
	})
}

// GetSubscriptions возвращает список подписок текущего пользователя
func (sc *SubscriptionController) GetSubscriptions(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := sc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(SubscriptionResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем параметры пагинации
	page, limit := sc.getPaginationParams(c)

	// Получаем подписки с информацией о пользователях
	var subscriptions []models.Subscription
	var total int64

	// Подсчитываем общее количество
	sc.DB.Model(&models.Subscription{}).Where("subscriber_id = ?", userID).Count(&total)

	// Получаем подписки с пагинацией
	offset := (page - 1) * limit
	err = sc.DB.Preload("SubscribedTo").
		Where("subscriber_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&subscriptions).Error

	if err != nil {
		return c.Status(500).JSON(SubscriptionResponse{
			Success: false,
			Message: "Ошибка при получении подписок",
		})
	}

	return c.JSON(SubscriptionResponse{
		Success: true,
		Message: "Подписки получены успешно",
		Data: fiber.Map{
			"subscriptions": subscriptions,
			"total":         total,
			"page":          page,
			"limit":         limit,
		},
	})
}

// GetSubscribers возвращает список подписчиков текущего пользователя
func (sc *SubscriptionController) GetSubscribers(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := sc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(SubscriptionResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем параметры пагинации
	page, limit := sc.getPaginationParams(c)

	// Получаем подписчиков с информацией о пользователях
	var subscriptions []models.Subscription
	var total int64

	// Подсчитываем общее количество
	sc.DB.Model(&models.Subscription{}).Where("subscribed_to_id = ?", userID).Count(&total)

	// Получаем подписчиков с пагинацией
	offset := (page - 1) * limit
	err = sc.DB.Preload("Subscriber").
		Where("subscribed_to_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&subscriptions).Error

	if err != nil {
		return c.Status(500).JSON(SubscriptionResponse{
			Success: false,
			Message: "Ошибка при получении подписчиков",
		})
	}

	return c.JSON(SubscriptionResponse{
		Success: true,
		Message: "Подписчики получены успешно",
		Data: fiber.Map{
			"subscribers": subscriptions,
			"total":       total,
			"page":        page,
			"limit":       limit,
		},
	})
}

// GetSubscriptionStats возвращает статистику подписок пользователя
func (sc *SubscriptionController) GetSubscriptionStats(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := sc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(SubscriptionResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Подсчитываем количество подписок
	var subscriptionsCount int64
	sc.DB.Model(&models.Subscription{}).Where("subscriber_id = ?", userID).Count(&subscriptionsCount)

	// Подсчитываем количество подписчиков
	var subscribersCount int64
	sc.DB.Model(&models.Subscription{}).Where("subscribed_to_id = ?", userID).Count(&subscribersCount)

	stats := models.SubscriptionStats{
		SubscriptionsCount: int(subscriptionsCount),
		SubscribersCount:   int(subscribersCount),
	}

	return c.JSON(SubscriptionResponse{
		Success: true,
		Message: "Статистика получена успешно",
		Data:    stats,
	})
}

// CheckSubscription проверяет, подписан ли текущий пользователь на указанного
func (sc *SubscriptionController) CheckSubscription(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := sc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(SubscriptionResponse{
			Success: false,
			Message: "Необходима авторизация",
		})
	}

	// Получаем ID пользователя для проверки из параметров URL
	targetUserIDStr := c.Params("user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		return c.Status(400).JSON(SubscriptionResponse{
			Success: false,
			Message: "Неверный ID пользователя",
		})
	}

	// Проверяем подписку
	var subscription models.Subscription
	err = sc.DB.Where("subscriber_id = ? AND subscribed_to_id = ?", userID, targetUserID).First(&subscription).Error
	isSubscribed := err == nil

	return c.JSON(SubscriptionResponse{
		Success: true,
		Message: "Статус подписки получен",
		Data: fiber.Map{
			"is_subscribed": isSubscribed,
		},
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (sc *SubscriptionController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
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
func (sc *SubscriptionController) getPaginationParams(c *fiber.Ctx) (int, int) {
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
