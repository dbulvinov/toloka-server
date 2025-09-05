package controllers

import (
	"strconv"
	"strings"

	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// ComplaintController контроллер для управления жалобами
type ComplaintController struct {
	DB *gorm.DB
}

// NewComplaintController создает новый экземпляр ComplaintController
func NewComplaintController(db *gorm.DB) *ComplaintController {
	return &ComplaintController{DB: db}
}

// SubmitComplaintRequest структура запроса подачи жалобы
type SubmitComplaintRequest struct {
	AboutUserID uint   `json:"about_user_id" validate:"required"`
	ReasonCode  string `json:"reason_code" validate:"required"`
	ReasonText  string `json:"reason_text" validate:"required,min=10,max=1000"`
}

// UpdateComplaintStatusRequest структура запроса обновления статуса жалобы
type UpdateComplaintStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=open under_review resolved dismissed"`
}

// ComplaintResponse структура ответа с жалобой
type ComplaintResponse struct {
	Success   bool              `json:"success"`
	Message   string            `json:"message"`
	Complaint *models.Complaint `json:"complaint,omitempty"`
}

// ComplaintsResponse структура ответа со списком жалоб
type ComplaintsResponse struct {
	Success    bool               `json:"success"`
	Message    string             `json:"message"`
	Complaints []models.Complaint `json:"complaints,omitempty"`
	Total      int64              `json:"total,omitempty"`
}

// ComplaintReasonsResponse структура ответа со списком причин жалоб
type ComplaintReasonsResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Reasons map[string]string `json:"reasons,omitempty"`
}

// SubmitComplaint подает жалобу на участника
func (cc *ComplaintController) SubmitComplaint(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ComplaintResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ComplaintResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента
	var event models.Event
	if err := cc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(ComplaintResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	var req SubmitComplaintRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ComplaintResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация запроса
	if err := cc.validateComplaintRequest(&req, uint(eventID), userID); err != nil {
		return c.Status(400).JSON(ComplaintResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Проверяем, не подавал ли уже пользователь жалобу на этого участника в этом ивенте
	var existingComplaint models.Complaint
	if err := cc.DB.Where("event_id = ? AND from_user_id = ? AND about_user_id = ?",
		eventID, userID, req.AboutUserID).First(&existingComplaint).Error; err == nil {
		return c.Status(409).JSON(ComplaintResponse{
			Success: false,
			Message: "Вы уже подали жалобу на этого участника в данном ивенте",
		})
	}

	// Создаем жалобу
	complaint := models.Complaint{
		EventID:     uint(eventID),
		FromUserID:  userID,
		AboutUserID: req.AboutUserID,
		ReasonCode:  req.ReasonCode,
		ReasonText:  req.ReasonText,
		Status:      models.ComplaintStatusOpen,
	}

	if err := cc.DB.Create(&complaint).Error; err != nil {
		return c.Status(500).JSON(ComplaintResponse{
			Success: false,
			Message: "Ошибка при создании жалобы",
		})
	}

	// Загружаем полную информацию о жалобе
	if err := cc.DB.Preload("FromUser").Preload("AboutUser").Preload("Event").First(&complaint, complaint.ID).Error; err != nil {
		return c.Status(500).JSON(ComplaintResponse{
			Success: false,
			Message: "Ошибка при загрузке данных жалобы",
		})
	}

	return c.Status(201).JSON(ComplaintResponse{
		Success:   true,
		Message:   "Жалоба успешно подана",
		Complaint: &complaint,
	})
}

// GetComplaints получает список жалоб (только для модераторов/админов)
func (cc *ComplaintController) GetComplaints(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ComplaintsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Проверяем права модератора (пока простая проверка - в реальном приложении нужна система ролей)
	if !cc.isModerator(userID) {
		return c.Status(403).JSON(ComplaintsResponse{
			Success: false,
			Message: "Нет прав для просмотра жалоб",
		})
	}

	// Получаем параметры запроса
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	status := c.Query("status")
	eventID := c.Query("event_id")

	// Валидация параметров
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Строим запрос
	query := cc.DB.Model(&models.Complaint{}).
		Preload("FromUser").
		Preload("AboutUser").
		Preload("Event")

	// Фильтр по статусу
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Фильтр по ивенту
	if eventID != "" {
		if eventIDUint, err := strconv.ParseUint(eventID, 10, 32); err == nil {
			query = query.Where("event_id = ?", eventIDUint)
		}
	}

	// Получаем общее количество
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(ComplaintsResponse{
			Success: false,
			Message: "Ошибка при получении количества жалоб",
		})
	}

	// Получаем жалобы
	var complaints []models.Complaint
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&complaints).Error; err != nil {
		return c.Status(500).JSON(ComplaintsResponse{
			Success: false,
			Message: "Ошибка при получении списка жалоб",
		})
	}

	return c.JSON(ComplaintsResponse{
		Success:    true,
		Message:    "Список жалоб получен",
		Complaints: complaints,
		Total:      total,
	})
}

// UpdateComplaintStatus обновляет статус жалобы
func (cc *ComplaintController) UpdateComplaintStatus(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ComplaintResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Проверяем права модератора
	if !cc.isModerator(userID) {
		return c.Status(403).JSON(ComplaintResponse{
			Success: false,
			Message: "Нет прав для изменения статуса жалобы",
		})
	}

	// Получаем ID жалобы
	complaintID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(ComplaintResponse{
			Success: false,
			Message: "Неверный ID жалобы",
		})
	}

	// Находим жалобу
	var complaint models.Complaint
	if err := cc.DB.First(&complaint, complaintID).Error; err != nil {
		return c.Status(404).JSON(ComplaintResponse{
			Success: false,
			Message: "Жалоба не найдена",
		})
	}

	var req UpdateComplaintStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ComplaintResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация статуса
	if !cc.isValidStatus(req.Status) {
		return c.Status(400).JSON(ComplaintResponse{
			Success: false,
			Message: "Неверный статус жалобы",
		})
	}

	// Обновляем статус
	complaint.Status = req.Status
	if err := cc.DB.Save(&complaint).Error; err != nil {
		return c.Status(500).JSON(ComplaintResponse{
			Success: false,
			Message: "Ошибка при обновлении статуса жалобы",
		})
	}

	// Загружаем полную информацию о жалобе
	if err := cc.DB.Preload("FromUser").Preload("AboutUser").Preload("Event").First(&complaint, complaint.ID).Error; err != nil {
		return c.Status(500).JSON(ComplaintResponse{
			Success: false,
			Message: "Ошибка при загрузке данных жалобы",
		})
	}

	return c.JSON(ComplaintResponse{
		Success:   true,
		Message:   "Статус жалобы обновлен",
		Complaint: &complaint,
	})
}

// GetComplaintReasons получает список доступных причин жалоб
func (cc *ComplaintController) GetComplaintReasons(c *fiber.Ctx) error {
	reasons := models.GetComplaintReasons()

	return c.JSON(ComplaintReasonsResponse{
		Success: true,
		Message: "Список причин жалоб получен",
		Reasons: reasons,
	})
}

// GetUserComplaints получает жалобы пользователя
func (cc *ComplaintController) GetUserComplaints(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := cc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(ComplaintsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем параметры запроса
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	status := c.Query("status")

	// Валидация параметров
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Строим запрос
	query := cc.DB.Model(&models.Complaint{}).
		Where("from_user_id = ?", userID).
		Preload("AboutUser").
		Preload("Event")

	// Фильтр по статусу
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Получаем общее количество
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(500).JSON(ComplaintsResponse{
			Success: false,
			Message: "Ошибка при получении количества жалоб",
		})
	}

	// Получаем жалобы
	var complaints []models.Complaint
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&complaints).Error; err != nil {
		return c.Status(500).JSON(ComplaintsResponse{
			Success: false,
			Message: "Ошибка при получении списка жалоб",
		})
	}

	return c.JSON(ComplaintsResponse{
		Success:    true,
		Message:    "Список ваших жалоб получен",
		Complaints: complaints,
		Total:      total,
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (cc *ComplaintController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
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

// validateComplaintRequest валидирует запрос жалобы
func (cc *ComplaintController) validateComplaintRequest(req *SubmitComplaintRequest, eventID, fromUserID uint) error {
	// Проверяем, что пользователь не жалуется сам на себя
	if req.AboutUserID == fromUserID {
		return fiber.NewError(400, "Нельзя подать жалобу на самого себя")
	}

	// Проверяем, что целевой пользователь участвовал в ивенте
	var participant models.EventParticipant
	if err := cc.DB.Where("event_id = ? AND user_id = ? AND status IN ?",
		eventID, req.AboutUserID, []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted}).
		First(&participant).Error; err != nil {
		return fiber.NewError(400, "Пользователь не участвовал в этом ивенте")
	}

	// Проверяем, что жалующийся тоже участвовал в ивенте
	if err := cc.DB.Where("event_id = ? AND user_id = ? AND status IN ?",
		eventID, fromUserID, []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted}).
		First(&participant).Error; err != nil {
		return fiber.NewError(400, "Вы не участвовали в этом ивенте")
	}

	// Проверяем код причины
	reasons := models.GetComplaintReasons()
	if _, exists := reasons[req.ReasonCode]; !exists {
		return fiber.NewError(400, "Неверный код причины жалобы")
	}

	// Проверяем длину текста жалобы
	if len(strings.TrimSpace(req.ReasonText)) < 10 {
		return fiber.NewError(400, "Текст жалобы должен содержать минимум 10 символов")
	}

	if len(req.ReasonText) > 1000 {
		return fiber.NewError(400, "Текст жалобы не должен превышать 1000 символов")
	}

	return nil
}

// isModerator проверяет, является ли пользователь модератором
func (cc *ComplaintController) isModerator(userID uint) bool {
	// В реальном приложении здесь должна быть проверка ролей пользователя
	// Пока используем простую проверку - пользователь с ID 1 считается модератором
	return userID == 1
}

// isValidStatus проверяет, является ли статус валидным
func (cc *ComplaintController) isValidStatus(status string) bool {
	validStatuses := []string{
		models.ComplaintStatusOpen,
		models.ComplaintStatusUnderReview,
		models.ComplaintStatusResolved,
		models.ComplaintStatusDismissed,
	}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}

	return false
}
