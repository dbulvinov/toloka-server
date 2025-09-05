package controllers

import (
	"strconv"
	"strings"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CommunityController контроллер для работы с сообществами
type CommunityController struct {
	DB *gorm.DB
}

// NewCommunityController создает новый экземпляр CommunityController
func NewCommunityController(db *gorm.DB) *CommunityController {
	return &CommunityController{DB: db}
}

// CreateCommunityRequest структура запроса создания сообщества
type CreateCommunityRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"max=500"`
	City        string `json:"city" validate:"required,min=2,max=100"`
}

// UpdateCommunityRequest структура запроса обновления сообщества
type UpdateCommunityRequest struct {
	Name        string `json:"name" validate:"min=3,max=100"`
	Description string `json:"description" validate:"max=500"`
	City        string `json:"city" validate:"min=2,max=100"`
}

// CommunityResponse структура ответа с сообществом
type CommunityResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    *models.Community `json:"data,omitempty"`
}

// CommunitiesResponse структура ответа со списком сообществ
type CommunitiesResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message"`
	Data    []models.Community `json:"data,omitempty"`
	Total   int64              `json:"total,omitempty"`
}

// CreateCommunity создает новое сообщество
func (cc *CommunityController) CreateCommunity(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(CommunityResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	var req CreateCommunityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := cc.validateCreateCommunityRequest(&req); err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Создаем сообщество
	community := models.Community{
		CreatorID:   userID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		City:        strings.TrimSpace(req.City),
	}

	if err := cc.DB.Create(&community).Error; err != nil {
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при создании сообщества",
		})
	}

	// Создаем роль администратора для создателя
	role := models.CommunityRole{
		CommunityID: community.ID,
		UserID:      userID,
		Role:        "admin",
	}

	if err := cc.DB.Create(&role).Error; err != nil {
		// Если не удалось создать роль, удаляем сообщество
		cc.DB.Delete(&community)
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при назначении роли",
		})
	}

	// Загружаем сообщество с создателем
	cc.DB.Preload("Creator").First(&community, community.ID)

	return c.Status(201).JSON(CommunityResponse{
		Success: true,
		Message: "Сообщество успешно создано",
		Data:    &community,
	})
}

// GetCommunities получает список сообществ с фильтрацией по городу
func (cc *CommunityController) GetCommunities(c *fiber.Ctx) error {
	// Получаем параметры запроса
	city := c.Query("city", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Строим запрос
	query := cc.DB.Model(&models.Community{}).Preload("Creator")

	if city != "" {
		query = query.Where("city ILIKE ?", "%"+city+"%")
	}

	// Получаем общее количество
	var total int64
	query.Count(&total)

	// Получаем сообщества
	var communities []models.Community
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&communities).Error; err != nil {
		return c.Status(500).JSON(CommunitiesResponse{
			Success: false,
			Message: "Ошибка при получении списка сообществ",
		})
	}

	return c.JSON(CommunitiesResponse{
		Success: true,
		Message: "Список сообществ получен",
		Data:    communities,
		Total:   total,
	})
}

// GetCommunity получает сообщество по ID
func (cc *CommunityController) GetCommunity(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: "Неверный ID сообщества",
		})
	}

	var community models.Community
	if err := cc.DB.Preload("Creator").Preload("Roles.User").First(&community, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommunityResponse{
				Success: false,
				Message: "Сообщество не найдено",
			})
		}
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при получении сообщества",
		})
	}

	return c.JSON(CommunityResponse{
		Success: true,
		Message: "Сообщество получено",
		Data:    &community,
	})
}

// UpdateCommunity обновляет сообщество
func (cc *CommunityController) UpdateCommunity(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(CommunityResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: "Неверный ID сообщества",
		})
	}

	var req UpdateCommunityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Проверяем существование сообщества
	var community models.Community
	if err := cc.DB.First(&community, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommunityResponse{
				Success: false,
				Message: "Сообщество не найдено",
			})
		}
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при получении сообщества",
		})
	}

	// Проверяем права доступа
	if !cc.canManageCommunity(userID, community.ID) {
		return c.Status(403).JSON(CommunityResponse{
			Success: false,
			Message: "Недостаточно прав для редактирования сообщества",
		})
	}

	// Обновляем поля
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		updates["description"] = strings.TrimSpace(req.Description)
	}
	if req.City != "" {
		updates["city"] = strings.TrimSpace(req.City)
	}

	if len(updates) == 0 {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: "Нет данных для обновления",
		})
	}

	// Валидация
	if err := cc.validateUpdateCommunityRequest(&req); err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Обновляем сообщество
	if err := cc.DB.Model(&community).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при обновлении сообщества",
		})
	}

	// Загружаем обновленное сообщество
	cc.DB.Preload("Creator").First(&community, community.ID)

	return c.JSON(CommunityResponse{
		Success: true,
		Message: "Сообщество успешно обновлено",
		Data:    &community,
	})
}

// DeleteCommunity удаляет сообщество
func (cc *CommunityController) DeleteCommunity(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(CommunityResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(CommunityResponse{
			Success: false,
			Message: "Неверный ID сообщества",
		})
	}

	// Проверяем существование сообщества
	var community models.Community
	if err := cc.DB.First(&community, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(CommunityResponse{
				Success: false,
				Message: "Сообщество не найдено",
			})
		}
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при получении сообщества",
		})
	}

	// Проверяем права доступа (только создатель может удалить)
	if community.CreatorID != userID {
		return c.Status(403).JSON(CommunityResponse{
			Success: false,
			Message: "Только создатель может удалить сообщество",
		})
	}

	// Удаляем сообщество (каскадное удаление)
	if err := cc.DB.Delete(&community).Error; err != nil {
		return c.Status(500).JSON(CommunityResponse{
			Success: false,
			Message: "Ошибка при удалении сообщества",
		})
	}

	return c.JSON(CommunityResponse{
		Success: true,
		Message: "Сообщество успешно удалено",
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (cc *CommunityController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
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

// canManageCommunity проверяет, может ли пользователь управлять сообществом
func (cc *CommunityController) canManageCommunity(userID, communityID uint) bool {
	var role models.CommunityRole
	err := cc.DB.Where("community_id = ? AND user_id = ? AND role IN ?",
		communityID, userID, []string{"admin", "moderator"}).First(&role).Error
	return err == nil
}

// validateCreateCommunityRequest валидирует запрос создания сообщества
func (cc *CommunityController) validateCreateCommunityRequest(req *CreateCommunityRequest) error {
	if req.Name == "" {
		return fiber.NewError(400, "Название сообщества обязательно")
	}
	if len(req.Name) < 3 || len(req.Name) > 100 {
		return fiber.NewError(400, "Название должно содержать от 3 до 100 символов")
	}
	if len(req.Description) > 500 {
		return fiber.NewError(400, "Описание не должно превышать 500 символов")
	}
	if req.City == "" {
		return fiber.NewError(400, "Город обязателен")
	}
	if len(req.City) < 2 || len(req.City) > 100 {
		return fiber.NewError(400, "Название города должно содержать от 2 до 100 символов")
	}
	return nil
}

// validateUpdateCommunityRequest валидирует запрос обновления сообщества
func (cc *CommunityController) validateUpdateCommunityRequest(req *UpdateCommunityRequest) error {
	if req.Name != "" && (len(req.Name) < 3 || len(req.Name) > 100) {
		return fiber.NewError(400, "Название должно содержать от 3 до 100 символов")
	}
	if req.Description != "" && len(req.Description) > 500 {
		return fiber.NewError(400, "Описание не должно превышать 500 символов")
	}
	if req.City != "" && (len(req.City) < 2 || len(req.City) > 100) {
		return fiber.NewError(400, "Название города должно содержать от 2 до 100 символов")
	}
	return nil
}
