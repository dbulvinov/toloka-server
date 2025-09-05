package controllers

import (
	"strconv"
	"strings"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// NewsController контроллер для работы с новостями
type NewsController struct {
	DB *gorm.DB
}

// NewNewsController создает новый экземпляр NewsController
func NewNewsController(db *gorm.DB) *NewsController {
	return &NewsController{DB: db}
}

// CreateNewsRequest структура запроса создания новости
type CreateNewsRequest struct {
	Content   string `json:"content" validate:"required,min=10,max=2000"`
	PhotoPath string `json:"photo_path" validate:"max=255"`
}

// UpdateNewsRequest структура запроса обновления новости
type UpdateNewsRequest struct {
	Content   string `json:"content" validate:"min=10,max=2000"`
	PhotoPath string `json:"photo_path" validate:"max=255"`
}

// NewsResponse структура ответа с новостью
type NewsResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    *models.News `json:"data,omitempty"`
}

// NewsListResponse структура ответа со списком новостей
type NewsListResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Data    []models.News `json:"data,omitempty"`
	Total   int64         `json:"total,omitempty"`
}

// CreateNews создает новую новость в сообществе
func (nc *NewsController) CreateNews(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := nc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(NewsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID сообщества
	communityID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный ID сообщества",
		})
	}

	var req CreateNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := nc.validateCreateNewsRequest(&req); err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Проверяем существование сообщества
	var community models.Community
	if err := nc.DB.First(&community, uint(communityID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(NewsResponse{
				Success: false,
				Message: "Сообщество не найдено",
			})
		}
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при получении сообщества",
		})
	}

	// Проверяем права доступа (только участники сообщества могут создавать новости)
	if !nc.canManageNews(userID, uint(communityID)) {
		return c.Status(403).JSON(NewsResponse{
			Success: false,
			Message: "Недостаточно прав для создания новости",
		})
	}

	// Создаем новость
	news := models.News{
		CommunityID: uint(communityID),
		AuthorID:    userID,
		Content:     strings.TrimSpace(req.Content),
		PhotoPath:   strings.TrimSpace(req.PhotoPath),
		LikesCount:  0,
	}

	if err := nc.DB.Create(&news).Error; err != nil {
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при создании новости",
		})
	}

	// Загружаем новость с автором и сообществом
	nc.DB.Preload("Author").Preload("Community").First(&news, news.ID)

	return c.Status(201).JSON(NewsResponse{
		Success: true,
		Message: "Новость успешно создана",
		Data:    &news,
	})
}

// GetNews получает список новостей сообщества
func (nc *NewsController) GetNews(c *fiber.Ctx) error {
	// Получаем ID сообщества
	communityID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(NewsListResponse{
			Success: false,
			Message: "Неверный ID сообщества",
		})
	}

	// Получаем параметры запроса
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Проверяем существование сообщества
	var community models.Community
	if err := nc.DB.First(&community, uint(communityID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(NewsListResponse{
				Success: false,
				Message: "Сообщество не найдено",
			})
		}
		return c.Status(500).JSON(NewsListResponse{
			Success: false,
			Message: "Ошибка при получении сообщества",
		})
	}

	// Получаем общее количество новостей
	var total int64
	nc.DB.Model(&models.News{}).Where("community_id = ?", communityID).Count(&total)

	// Получаем новости
	var news []models.News
	query := nc.DB.Where("community_id = ?", communityID).
		Preload("Author").
		Preload("Community").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Where("parent_id IS NULL").Order("created_at ASC")
		}).
		Preload("Comments.Children", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Order("created_at ASC")
		})

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&news).Error; err != nil {
		return c.Status(500).JSON(NewsListResponse{
			Success: false,
			Message: "Ошибка при получении новостей",
		})
	}

	return c.JSON(NewsListResponse{
		Success: true,
		Message: "Список новостей получен",
		Data:    news,
		Total:   total,
	})
}

// GetNewsByID получает новость по ID
func (nc *NewsController) GetNewsByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный ID новости",
		})
	}

	var news models.News
	if err := nc.DB.Preload("Author").
		Preload("Community").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Where("parent_id IS NULL").Order("created_at ASC")
		}).
		Preload("Comments.Children", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Order("created_at ASC")
		}).
		Preload("Likes.User").
		First(&news, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(NewsResponse{
				Success: false,
				Message: "Новость не найдена",
			})
		}
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при получении новости",
		})
	}

	return c.JSON(NewsResponse{
		Success: true,
		Message: "Новость получена",
		Data:    &news,
	})
}

// UpdateNews обновляет новость
func (nc *NewsController) UpdateNews(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := nc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(NewsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный ID новости",
		})
	}

	var req UpdateNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Проверяем существование новости
	var news models.News
	if err := nc.DB.First(&news, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(NewsResponse{
				Success: false,
				Message: "Новость не найдена",
			})
		}
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при получении новости",
		})
	}

	// Проверяем права доступа (автор или модератор/админ сообщества)
	if news.AuthorID != userID && !nc.canManageNews(userID, news.CommunityID) {
		return c.Status(403).JSON(NewsResponse{
			Success: false,
			Message: "Недостаточно прав для редактирования новости",
		})
	}

	// Валидация
	if err := nc.validateUpdateNewsRequest(&req); err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Обновляем поля
	updates := make(map[string]interface{})
	if req.Content != "" {
		updates["content"] = strings.TrimSpace(req.Content)
	}
	if req.PhotoPath != "" {
		updates["photo_path"] = strings.TrimSpace(req.PhotoPath)
	}

	if len(updates) == 0 {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Нет данных для обновления",
		})
	}

	// Обновляем новость
	if err := nc.DB.Model(&news).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при обновлении новости",
		})
	}

	// Загружаем обновленную новость
	nc.DB.Preload("Author").Preload("Community").First(&news, news.ID)

	return c.JSON(NewsResponse{
		Success: true,
		Message: "Новость успешно обновлена",
		Data:    &news,
	})
}

// DeleteNews удаляет новость
func (nc *NewsController) DeleteNews(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := nc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(NewsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный ID новости",
		})
	}

	// Проверяем существование новости
	var news models.News
	if err := nc.DB.First(&news, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(NewsResponse{
				Success: false,
				Message: "Новость не найдена",
			})
		}
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при получении новости",
		})
	}

	// Проверяем права доступа (автор или модератор/админ сообщества)
	if news.AuthorID != userID && !nc.canManageNews(userID, news.CommunityID) {
		return c.Status(403).JSON(NewsResponse{
			Success: false,
			Message: "Недостаточно прав для удаления новости",
		})
	}

	// Удаляем новость (каскадное удаление)
	if err := nc.DB.Delete(&news).Error; err != nil {
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при удалении новости",
		})
	}

	return c.JSON(NewsResponse{
		Success: true,
		Message: "Новость успешно удалена",
	})
}

// LikeNews добавляет/убирает лайк новости
func (nc *NewsController) LikeNews(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := nc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(NewsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(NewsResponse{
			Success: false,
			Message: "Неверный ID новости",
		})
	}

	// Проверяем существование новости
	var news models.News
	if err := nc.DB.First(&news, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(NewsResponse{
				Success: false,
				Message: "Новость не найдена",
			})
		}
		return c.Status(500).JSON(NewsResponse{
			Success: false,
			Message: "Ошибка при получении новости",
		})
	}

	// Проверяем, есть ли уже лайк от этого пользователя
	var existingLike models.NewsLike
	err = nc.DB.Where("news_id = ? AND user_id = ?", id, userID).First(&existingLike).Error

	if err == gorm.ErrRecordNotFound {
		// Создаем новый лайк
		like := models.NewsLike{
			NewsID: uint(id),
			UserID: userID,
		}

		if err := nc.DB.Create(&like).Error; err != nil {
			return c.Status(500).JSON(NewsResponse{
				Success: false,
				Message: "Ошибка при добавлении лайка",
			})
		}

		// Увеличиваем счетчик лайков
		nc.DB.Model(&news).Update("likes_count", gorm.Expr("likes_count + 1"))

		return c.JSON(NewsResponse{
			Success: true,
			Message: "Лайк добавлен",
		})
	} else if err == nil {
		// Удаляем существующий лайк
		if err := nc.DB.Delete(&existingLike).Error; err != nil {
			return c.Status(500).JSON(NewsResponse{
				Success: false,
				Message: "Ошибка при удалении лайка",
			})
		}

		// Уменьшаем счетчик лайков
		nc.DB.Model(&news).Update("likes_count", gorm.Expr("likes_count - 1"))

		return c.JSON(NewsResponse{
			Success: true,
			Message: "Лайк удален",
		})
	}

	return c.Status(500).JSON(NewsResponse{
		Success: false,
		Message: "Ошибка при обработке лайка",
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (nc *NewsController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0, fiber.NewError(401, "Отсутствует токен авторизации")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return 0, fiber.NewError(401, "Неверный формат токена")
	}

	claims, err := utils.ValidateJWT(tokenString)
	if err != nil {
		return 0, fiber.NewError(401, "Недействительный токен")
	}

	return claims.UserID, nil
}

// canManageNews проверяет, может ли пользователь управлять новостями в сообществе
func (nc *NewsController) canManageNews(userID, communityID uint) bool {
	var role models.CommunityRole
	err := nc.DB.Where("community_id = ? AND user_id = ? AND role IN ?",
		communityID, userID, []string{"admin", "moderator"}).First(&role).Error
	return err == nil
}

// validateCreateNewsRequest валидирует запрос создания новости
func (nc *NewsController) validateCreateNewsRequest(req *CreateNewsRequest) error {
	if req.Content == "" {
		return fiber.NewError(400, "Содержимое новости обязательно")
	}
	if len(req.Content) < 10 || len(req.Content) > 2000 {
		return fiber.NewError(400, "Содержимое должно содержать от 10 до 2000 символов")
	}
	if len(req.PhotoPath) > 255 {
		return fiber.NewError(400, "Путь к фото не должен превышать 255 символов")
	}
	return nil
}

// validateUpdateNewsRequest валидирует запрос обновления новости
func (nc *NewsController) validateUpdateNewsRequest(req *UpdateNewsRequest) error {
	if req.Content != "" && (len(req.Content) < 10 || len(req.Content) > 2000) {
		return fiber.NewError(400, "Содержимое должно содержать от 10 до 2000 символов")
	}
	if req.PhotoPath != "" && len(req.PhotoPath) > 255 {
		return fiber.NewError(400, "Путь к фото не должен превышать 255 символов")
	}
	return nil
}
