package controllers

import (
	"strconv"
	"strings"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CommentController контроллер для работы с комментариями
type CommentController struct {
	DB *gorm.DB
}

// NewCommentController создает новый экземпляр CommentController
func NewCommentController(db *gorm.DB) *CommentController {
	return &CommentController{DB: db}
}

// CreateCommentRequest структура запроса создания комментария
type CreateCommentRequest struct {
	Content  string `json:"content" validate:"required,min=1,max=1000"`
	ParentID *uint  `json:"parent_id"` // Для ответов на комментарии
}

// UpdateCommentRequest структура запроса обновления комментария
type UpdateCommentRequest struct {
	Content string `json:"content" validate:"min=1,max=1000"`
}

// CommentResponse структура ответа с комментарием
type CommentResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    *models.Comment `json:"data,omitempty"`
}

// CommentsResponse структура ответа со списком комментариев
type CommentsResponse struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Data    []models.Comment `json:"data,omitempty"`
	Total   int64            `json:"total,omitempty"`
}

// CreateComment создает новый комментарий к новости
func (cc *CommentController) CreateComment(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(CommentResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID новости
	newsID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Неверный ID новости",
		})
	}

	var req CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := cc.validateCreateCommentRequest(&req); err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Проверяем существование новости
	var news models.News
	if err := cc.DB.Preload("Community").First(&news, uint(newsID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommentResponse{
				Success: false,
				Message: "Новость не найдена",
			})
		}
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при получении новости",
		})
	}

	// Если указан parent_id, проверяем существование родительского комментария
	if req.ParentID != nil {
		var parentComment models.Comment
		if err := cc.DB.Where("id = ? AND news_id = ?", *req.ParentID, newsID).First(&parentComment).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(CommentResponse{
					Success: false,
					Message: "Родительский комментарий не найден",
				})
			}
			return c.Status(500).JSON(CommentResponse{
				Success: false,
				Message: "Ошибка при получении родительского комментария",
			})
		}
	}

	// Создаем комментарий
	comment := models.Comment{
		NewsID:   uint(newsID),
		AuthorID: userID,
		ParentID: req.ParentID,
		Content:  strings.TrimSpace(req.Content),
	}

	if err := cc.DB.Create(&comment).Error; err != nil {
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при создании комментария",
		})
	}

	// Загружаем комментарий с автором и новостью
	cc.DB.Preload("Author").Preload("News").Preload("Parent").First(&comment, comment.ID)

	return c.Status(201).JSON(CommentResponse{
		Success: true,
		Message: "Комментарий успешно создан",
		Data:    &comment,
	})
}

// GetComments получает список комментариев к новости
func (cc *CommentController) GetComments(c *fiber.Ctx) error {
	// Получаем ID новости
	newsID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommentsResponse{
			Success: false,
			Message: "Неверный ID новости",
		})
	}

	// Получаем параметры запроса
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	// Проверяем существование новости
	var news models.News
	if err := cc.DB.First(&news, uint(newsID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommentsResponse{
				Success: false,
				Message: "Новость не найдена",
			})
		}
		return c.Status(500).JSON(CommentsResponse{
			Success: false,
			Message: "Ошибка при получении новости",
		})
	}

	// Получаем общее количество комментариев
	var total int64
	cc.DB.Model(&models.Comment{}).Where("news_id = ?", newsID).Count(&total)

	// Получаем комментарии (только корневые, без ответов)
	var comments []models.Comment
	query := cc.DB.Where("news_id = ? AND parent_id IS NULL", newsID).
		Preload("Author").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Order("created_at ASC")
		})

	if err := query.Offset(offset).Limit(limit).Order("created_at ASC").Find(&comments).Error; err != nil {
		return c.Status(500).JSON(CommentsResponse{
			Success: false,
			Message: "Ошибка при получении комментариев",
		})
	}

	return c.JSON(CommentsResponse{
		Success: true,
		Message: "Список комментариев получен",
		Data:    comments,
		Total:   total,
	})
}

// GetComment получает комментарий по ID
func (cc *CommentController) GetComment(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Неверный ID комментария",
		})
	}

	var comment models.Comment
	if err := cc.DB.Preload("Author").
		Preload("News").
		Preload("Parent").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author").Order("created_at ASC")
		}).
		First(&comment, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommentResponse{
				Success: false,
				Message: "Комментарий не найден",
			})
		}
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при получении комментария",
		})
	}

	return c.JSON(CommentResponse{
		Success: true,
		Message: "Комментарий получен",
		Data:    &comment,
	})
}

// UpdateComment обновляет комментарий
func (cc *CommentController) UpdateComment(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(CommentResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Неверный ID комментария",
		})
	}

	var req UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Проверяем существование комментария
	var comment models.Comment
	if err := cc.DB.Preload("News.Community").First(&comment, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommentResponse{
				Success: false,
				Message: "Комментарий не найден",
			})
		}
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при получении комментария",
		})
	}

	// Проверяем права доступа (автор или модератор/админ сообщества)
	if comment.AuthorID != userID && !cc.canManageComments(userID, comment.News.CommunityID) {
		return c.Status(403).JSON(CommentResponse{
			Success: false,
			Message: "Недостаточно прав для редактирования комментария",
		})
	}

	// Валидация
	if err := cc.validateUpdateCommentRequest(&req); err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Обновляем содержимое
	if req.Content == "" {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Содержимое комментария обязательно",
		})
	}

	// Обновляем комментарий
	if err := cc.DB.Model(&comment).Update("content", strings.TrimSpace(req.Content)).Error; err != nil {
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при обновлении комментария",
		})
	}

	// Загружаем обновленный комментарий
	cc.DB.Preload("Author").Preload("News").Preload("Parent").First(&comment, comment.ID)

	return c.JSON(CommentResponse{
		Success: true,
		Message: "Комментарий успешно обновлен",
		Data:    &comment,
	})
}

// DeleteComment удаляет комментарий
func (cc *CommentController) DeleteComment(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(CommentResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommentResponse{
			Success: false,
			Message: "Неверный ID комментария",
		})
	}

	// Проверяем существование комментария
	var comment models.Comment
	if err := cc.DB.Preload("News.Community").First(&comment, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommentResponse{
				Success: false,
				Message: "Комментарий не найден",
			})
		}
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при получении комментария",
		})
	}

	// Проверяем права доступа (автор или модератор/админ сообщества)
	if comment.AuthorID != userID && !cc.canManageComments(userID, comment.News.CommunityID) {
		return c.Status(403).JSON(CommentResponse{
			Success: false,
			Message: "Недостаточно прав для удаления комментария",
		})
	}

	// Удаляем комментарий (каскадное удаление для дочерних комментариев)
	if err := cc.DB.Delete(&comment).Error; err != nil {
		return c.Status(500).JSON(CommentResponse{
			Success: false,
			Message: "Ошибка при удалении комментария",
		})
	}

	return c.JSON(CommentResponse{
		Success: true,
		Message: "Комментарий успешно удален",
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (cc *CommentController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
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

// canManageComments проверяет, может ли пользователь управлять комментариями в сообществе
func (cc *CommentController) canManageComments(userID, communityID uint) bool {
	var role models.CommunityRole
	err := cc.DB.Where("community_id = ? AND user_id = ? AND role IN ?",
		communityID, userID, []string{"admin", "moderator"}).First(&role).Error
	return err == nil
}

// validateCreateCommentRequest валидирует запрос создания комментария
func (cc *CommentController) validateCreateCommentRequest(req *CreateCommentRequest) error {
	if req.Content == "" {
		return fiber.NewError(400, "Содержимое комментария обязательно")
	}
	if len(req.Content) < 1 || len(req.Content) > 1000 {
		return fiber.NewError(400, "Содержимое должно содержать от 1 до 1000 символов")
	}
	return nil
}

// validateUpdateCommentRequest валидирует запрос обновления комментария
func (cc *CommentController) validateUpdateCommentRequest(req *UpdateCommentRequest) error {
	if req.Content != "" && (len(req.Content) < 1 || len(req.Content) > 1000) {
		return fiber.NewError(400, "Содержимое должно содержать от 1 до 1000 символов")
	}
	return nil
}
