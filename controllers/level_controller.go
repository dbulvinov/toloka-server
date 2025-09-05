package controllers

import (
	"strconv"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// LevelController контроллер для управления уровнями пользователей
type LevelController struct {
	db *gorm.DB
}

// NewLevelController создает новый экземпляр LevelController
func NewLevelController(db *gorm.DB) *LevelController {
	return &LevelController{db: db}
}

// GetUserLevel получает уровень пользователя
func (lc *LevelController) GetUserLevel(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID пользователя",
		})
	}

	// Проверяем авторизацию
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Необходима авторизация",
		})
	}

	claims, err := utils.ValidateJWT(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный токен авторизации",
		})
	}

	// Проверяем, что пользователь запрашивает свой уровень или имеет права
	if claims.UserID != uint(userID) {
		return c.Status(403).JSON(fiber.Map{
			"error":   true,
			"message": "Можно просматривать только свой уровень",
		})
	}

	// Получаем уровень пользователя
	var userLevel models.UserLevel
	if err := lc.db.Where("user_id = ?", userID).First(&userLevel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Создаем начальный уровень, если его нет
			userLevel = models.UserLevel{
				UserID: uint(userID),
				Level:  1,
				Points: 0,
			}
			if err := lc.db.Create(&userLevel).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error":   true,
					"message": "Ошибка при создании уровня пользователя",
				})
			}
		} else {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": "Ошибка при получении уровня пользователя",
			})
		}
	}

	// Вычисляем прогресс до следующего уровня
	currentLevelPoints := (userLevel.Level - 1) * 100
	nextLevelPoints := userLevel.Level * 100
	progressPoints := userLevel.Points - currentLevelPoints
	pointsToNextLevel := nextLevelPoints - userLevel.Points
	progressPercentage := float64(progressPoints) / float64(100) * 100

	// Получаем статистику пользователя
	var stats struct {
		EventsParticipated int64 `json:"events_participated"`
		EventsCreated      int64 `json:"events_created"`
		CommunitiesCreated int64 `json:"communities_created"`
		NewsCreated        int64 `json:"news_created"`
		CommentsCreated    int64 `json:"comments_created"`
	}

	// Подсчитываем статистику
	lc.db.Model(&models.EventParticipant{}).Where("user_id = ?", userID).Count(&stats.EventsParticipated)
	lc.db.Model(&models.Event{}).Where("created_by = ?", userID).Count(&stats.EventsCreated)
	lc.db.Model(&models.Community{}).Where("created_by = ?", userID).Count(&stats.CommunitiesCreated)
	lc.db.Model(&models.News{}).Where("created_by = ?", userID).Count(&stats.NewsCreated)
	lc.db.Model(&models.Comment{}).Where("user_id = ?", userID).Count(&stats.CommentsCreated)

	response := fiber.Map{
		"user_level": userLevel,
		"progress": fiber.Map{
			"current_level_points": currentLevelPoints,
			"next_level_points":    nextLevelPoints,
			"progress_points":      progressPoints,
			"points_to_next_level": pointsToNextLevel,
			"progress_percentage":  progressPercentage,
		},
		"stats":   stats,
		"error":   false,
		"message": "Уровень пользователя получен успешно",
	}

	return c.JSON(response)
}

// GetLeaderboard получает таблицу лидеров
func (lc *LevelController) GetLeaderboard(c *fiber.Ctx) error {
	// Получаем параметры пагинации
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	// Получаем топ пользователей по очкам
	var userLevels []models.UserLevel
	if err := lc.db.Preload("User").Order("points DESC, level DESC").Limit(limit).Offset(offset).Find(&userLevels).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении таблицы лидеров",
		})
	}

	// Получаем общее количество пользователей
	var totalCount int64
	lc.db.Model(&models.UserLevel{}).Count(&totalCount)

	// Формируем ответ
	var leaderboard []fiber.Map
	for i, ul := range userLevels {
		leaderboard = append(leaderboard, fiber.Map{
			"rank":   offset + i + 1,
			"user":   ul.User,
			"level":  ul.Level,
			"points": ul.Points,
		})
	}

	return c.JSON(fiber.Map{
		"leaderboard": leaderboard,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total_count": totalCount,
			"total_pages": (totalCount + int64(limit) - 1) / int64(limit),
		},
		"error":   false,
		"message": "Таблица лидеров получена успешно",
	})
}

// AddPoints добавляет очки пользователю
func (lc *LevelController) AddPoints(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID пользователя",
		})
	}

	// Проверяем авторизацию
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Необходима авторизация",
		})
	}

	claims, err := utils.ValidateJWT(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный токен авторизации",
		})
	}

	// Проверяем, что пользователь добавляет очки себе или имеет права
	if claims.UserID != uint(userID) {
		return c.Status(403).JSON(fiber.Map{
			"error":   true,
			"message": "Можно добавлять очки только себе",
		})
	}

	var requestData struct {
		Points int    `json:"points"`
		Reason string `json:"reason"`
	}

	if err := c.BodyParser(&requestData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный формат данных",
		})
	}

	if requestData.Points <= 0 {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Количество очков должно быть положительным",
		})
	}

	// Получаем или создаем уровень пользователя
	var userLevel models.UserLevel
	if err := lc.db.Where("user_id = ?", userID).First(&userLevel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			userLevel = models.UserLevel{
				UserID: uint(userID),
				Level:  1,
				Points: requestData.Points,
			}
			if err := lc.db.Create(&userLevel).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error":   true,
					"message": "Ошибка при создании уровня пользователя",
				})
			}
		} else {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": "Ошибка при получении уровня пользователя",
			})
		}
	} else {
		// Обновляем существующий уровень
		userLevel.Points += requestData.Points

		// Проверяем, нужно ли повысить уровень
		newLevel := (userLevel.Points / 100) + 1
		if newLevel > userLevel.Level {
			userLevel.Level = newLevel
		}

		if err := lc.db.Save(&userLevel).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   true,
				"message": "Ошибка при обновлении уровня пользователя",
			})
		}
	}

	return c.JSON(fiber.Map{
		"user_level":   userLevel,
		"added_points": requestData.Points,
		"reason":       requestData.Reason,
		"error":        false,
		"message":      "Очки успешно добавлены",
	})
}
