package controllers

import (
	"strconv"
	"time"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// UserController контроллер для управления профилями пользователей
type UserController struct {
	db *gorm.DB
}

// NewUserController создает новый экземпляр UserController
func NewUserController(db *gorm.DB) *UserController {
	return &UserController{db: db}
}

// GetProfile получает профиль пользователя по ID
func (uc *UserController) GetProfile(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID пользователя",
		})
	}

	var user models.User
	if err := uc.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "Пользователь не найден",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении профиля",
		})
	}

	// Если профиль не публичный, проверяем авторизацию
	if !user.IsPublic {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(401).JSON(fiber.Map{
				"error":   true,
				"message": "Необходима авторизация",
			})
		}

		// Проверяем JWT токен
		claims, err := utils.ValidateJWT(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error":   true,
				"message": "Неверный токен авторизации",
			})
		}

		// Проверяем, что пользователь запрашивает свой профиль или имеет права
		if claims.UserID != uint(userID) {
			return c.Status(403).JSON(fiber.Map{
				"error":   true,
				"message": "Доступ запрещен",
			})
		}
	}

	// Получаем статистику пользователя
	var stats struct {
		EventsCount      int64 `json:"events_count"`
		CommunitiesCount int64 `json:"communities_count"`
		FollowersCount   int64 `json:"followers_count"`
		FollowingCount   int64 `json:"following_count"`
	}

	// Подсчитываем количество событий пользователя
	uc.db.Model(&models.Event{}).Where("created_by = ?", userID).Count(&stats.EventsCount)

	// Подсчитываем количество сообществ пользователя
	uc.db.Model(&models.Community{}).Where("created_by = ?", userID).Count(&stats.CommunitiesCount)

	// Подсчитываем подписчиков и подписки
	uc.db.Model(&models.Subscription{}).Where("target_user_id = ?", userID).Count(&stats.FollowersCount)
	uc.db.Model(&models.Subscription{}).Where("user_id = ?", userID).Count(&stats.FollowingCount)

	// Получаем уровень пользователя
	var userLevel models.UserLevel
	uc.db.Where("user_id = ?", userID).First(&userLevel)

	response := fiber.Map{
		"user":    user,
		"stats":   stats,
		"level":   userLevel,
		"error":   false,
		"message": "Профиль получен успешно",
	}

	return c.JSON(response)
}

// UpdateProfile обновляет профиль пользователя
func (uc *UserController) UpdateProfile(c *fiber.Ctx) error {
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

	// Проверяем, что пользователь обновляет свой профиль
	if claims.UserID != uint(userID) {
		return c.Status(403).JSON(fiber.Map{
			"error":   true,
			"message": "Можно обновлять только свой профиль",
		})
	}

	var user models.User
	if err := uc.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "Пользователь не найден",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении профиля",
		})
	}

	// Обновляем только разрешенные поля
	var updateData struct {
		Name     string `json:"name"`
		Avatar   string `json:"avatar"`
		Bio      string `json:"bio"`
		Location string `json:"location"`
		Website  string `json:"website"`
		IsPublic bool   `json:"is_public"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный формат данных",
		})
	}

	// Обновляем поля
	if updateData.Name != "" {
		user.Name = updateData.Name
	}
	user.Avatar = updateData.Avatar
	user.Bio = updateData.Bio
	user.Location = updateData.Location
	user.Website = updateData.Website
	user.IsPublic = updateData.IsPublic
	user.UpdatedAt = time.Now()

	if err := uc.db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при обновлении профиля",
		})
	}

	return c.JSON(fiber.Map{
		"user":    user,
		"error":   false,
		"message": "Профиль обновлен успешно",
	})
}

// GetUserEvents получает события пользователя
func (uc *UserController) GetUserEvents(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID пользователя",
		})
	}

	// Проверяем, что пользователь существует
	var user models.User
	if err := uc.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "Пользователь не найден",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении данных пользователя",
		})
	}

	// Если профиль не публичный, проверяем авторизацию
	if !user.IsPublic {
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

		if claims.UserID != uint(userID) {
			return c.Status(403).JSON(fiber.Map{
				"error":   true,
				"message": "Доступ запрещен",
			})
		}
	}

	var events []models.Event
	if err := uc.db.Where("created_by = ?", userID).Order("created_at DESC").Find(&events).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении событий",
		})
	}

	return c.JSON(fiber.Map{
		"events":  events,
		"error":   false,
		"message": "События получены успешно",
	})
}

// GetUserCommunities получает сообщества пользователя
func (uc *UserController) GetUserCommunities(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   true,
			"message": "Неверный ID пользователя",
		})
	}

	// Проверяем, что пользователь существует
	var user models.User
	if err := uc.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{
				"error":   true,
				"message": "Пользователь не найден",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении данных пользователя",
		})
	}

	// Если профиль не публичный, проверяем авторизацию
	if !user.IsPublic {
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

		if claims.UserID != uint(userID) {
			return c.Status(403).JSON(fiber.Map{
				"error":   true,
				"message": "Доступ запрещен",
			})
		}
	}

	// Получаем сообщества, созданные пользователем
	var communities []models.Community
	if err := uc.db.Where("created_by = ?", userID).Order("created_at DESC").Find(&communities).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении сообществ",
		})
	}

	// Получаем новости из сообществ пользователя
	var news []models.News
	if err := uc.db.Joins("JOIN communities ON news.community_id = communities.id").
		Where("communities.created_by = ?", userID).
		Order("news.created_at DESC").
		Find(&news).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Ошибка при получении новостей",
		})
	}

	return c.JSON(fiber.Map{
		"communities": communities,
		"news":        news,
		"error":       false,
		"message":     "Данные получены успешно",
	})
}
