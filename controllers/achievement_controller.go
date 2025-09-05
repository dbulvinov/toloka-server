package controllers

import (
	"strconv"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AchievementController контроллер для управления достижениями
type AchievementController struct {
	db *gorm.DB
}

// NewAchievementController создает новый экземпляр AchievementController
func NewAchievementController(db *gorm.DB) *AchievementController {
	return &AchievementController{db: db}
}

// GetUserAchievements получает достижения пользователя
func (ac *AchievementController) GetUserAchievements(c *fiber.Ctx) error {
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

	// Проверяем, что пользователь запрашивает свои достижения или имеет права
	if claims.UserID != uint(userID) {
		return c.Status(403).JSON(fiber.Map{
			"error":   true,
			"message": "Можно просматривать только свои достижения",
		})
	}

	// Получаем достижения пользователя
	var userAchievements []models.UserAchievement
	if err := ac.db.Preload("Achievement").Where("user_id = ?", userID).Order("earned_at DESC").Find(&userAchievements).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении достижений",
		})
	}

	// Получаем уровень пользователя
	var userLevel models.UserLevel
	ac.db.Where("user_id = ?", userID).First(&userLevel)

	// Получаем все доступные достижения для отображения прогресса
	var allAchievements []models.Achievement
	ac.db.Where("is_active = ?", true).Find(&allAchievements)

	// Создаем карту полученных достижений
	earnedMap := make(map[uint]bool)
	for _, ua := range userAchievements {
		earnedMap[ua.AchievementID] = true
	}

	// Формируем ответ с прогрессом
	var achievementsWithProgress []fiber.Map
	for _, achievement := range allAchievements {
		achievementsWithProgress = append(achievementsWithProgress, fiber.Map{
			"achievement": achievement,
			"earned":      earnedMap[achievement.ID],
			"earned_at":   nil,
		})
	}

	// Добавляем дату получения для заработанных достижений
	for _, ua := range userAchievements {
		for i, awp := range achievementsWithProgress {
			if awp["achievement"].(models.Achievement).ID == ua.AchievementID {
				achievementsWithProgress[i]["earned_at"] = ua.EarnedAt
				break
			}
		}
	}

	return c.JSON(fiber.Map{
		"achievements": achievementsWithProgress,
		"user_level":   userLevel,
		"error":        false,
		"message":      "Достижения получены успешно",
	})
}

// GetAllAchievements получает все доступные достижения
func (ac *AchievementController) GetAllAchievements(c *fiber.Ctx) error {
	var achievements []models.Achievement
	if err := ac.db.Where("is_active = ?", true).Order("category, name").Find(&achievements).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении достижений",
		})
	}

	return c.JSON(fiber.Map{
		"achievements": achievements,
		"error":        false,
		"message":      "Достижения получены успешно",
	})
}

// AwardAchievement награждает пользователя достижением
func (ac *AchievementController) AwardAchievement(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID пользователя",
		})
	}

	achievementID, err := strconv.ParseUint(c.Params("achievement_id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID достижения",
		})
	}

	// Проверяем авторизацию (только админы могут награждать достижениями)
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Необходима авторизация",
		})
	}

	_, err = utils.ValidateJWT(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный токен авторизации",
		})
	}

	// Проверяем, что пользователь существует
	var user models.User
	if err := ac.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "Пользователь не найден",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении пользователя",
		})
	}

	// Проверяем, что достижение существует
	var achievement models.Achievement
	if err := ac.db.First(&achievement, achievementID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "Достижение не найдено",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении достижения",
		})
	}

	// Проверяем, не получено ли уже это достижение
	var existingUserAchievement models.UserAchievement
	if err := ac.db.Where("user_id = ? AND achievement_id = ?", userID, achievementID).First(&existingUserAchievement).Error; err == nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Достижение уже получено",
		})
	}

	// Создаем запись о получении достижения
	userAchievement := models.UserAchievement{
		UserID:        uint(userID),
		AchievementID: uint(achievementID),
	}

	if err := ac.db.Create(&userAchievement).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при награждении достижением",
		})
	}

	// Обновляем уровень пользователя
	ac.updateUserLevel(uint(userID), achievement.Points)

	return c.JSON(fiber.Map{
		"user_achievement": userAchievement,
		"error":            false,
		"message":          "Достижение успешно получено",
	})
}

// updateUserLevel обновляет уровень пользователя
func (ac *AchievementController) updateUserLevel(userID uint, points int) {
	var userLevel models.UserLevel
	if err := ac.db.Where("user_id = ?", userID).First(&userLevel).Error; err != nil {
		// Создаем новый уровень, если его нет
		userLevel = models.UserLevel{
			UserID: userID,
			Level:  1,
			Points: points,
		}
		ac.db.Create(&userLevel)
	} else {
		// Обновляем существующий уровень
		userLevel.Points += points
		// Проверяем, нужно ли повысить уровень (каждые 100 очков = новый уровень)
		newLevel := (userLevel.Points / 100) + 1
		if newLevel > userLevel.Level {
			userLevel.Level = newLevel
		}
		ac.db.Save(&userLevel)
	}
}
