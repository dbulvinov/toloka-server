package controllers

import (
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// RatingController контроллер для управления рейтингами
type RatingController struct {
	DB *gorm.DB
}

// NewRatingController создает новый экземпляр RatingController
func NewRatingController(db *gorm.DB) *RatingController {
	return &RatingController{DB: db}
}

// SubmitRatingsRequest структура запроса отправки рейтингов
type SubmitRatingsRequest struct {
	Ratings []RatingRequest `json:"ratings" validate:"required,min=1"`
}

// RatingRequest структура запроса рейтинга
type RatingRequest struct {
	TargetUserID uint   `json:"target_user_id" validate:"required"`
	Score        int    `json:"score" validate:"required,min=1,max=10"`
	Comment      string `json:"comment" validate:"max=500"`
}

// UploadPhotosRequest структура запроса загрузки фотографий
type UploadPhotosRequest struct {
	// Фотографии будут обрабатываться как multipart/form-data
}

// RatingResponse структура ответа с рейтингом
type RatingResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Rating  *models.Rating `json:"rating,omitempty"`
}

// RatingsResponse структура ответа со списком рейтингов
type RatingsResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Ratings []models.Rating `json:"ratings,omitempty"`
	Total   int64           `json:"total,omitempty"`
}

// UserRatingResponse структура ответа с рейтингом пользователя
type UserRatingResponse struct {
	Success       bool                      `json:"success"`
	Message       string                    `json:"message"`
	RatingSummary *models.UserRatingSummary `json:"rating_summary,omitempty"`
}

// SubmitRatings отправляет рейтинги участников
func (rc *RatingController) SubmitRatings(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := rc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(RatingResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента
	var event models.Event
	if err := rc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(RatingResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	// Проверяем, что ивент завершен
	if event.IsActive {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Нельзя оценивать участников незавершенного ивента",
		})
	}

	// Проверяем, что пользователь участвовал в ивенте
	var participant models.EventParticipant
	if err := rc.DB.Where("event_id = ? AND user_id = ? AND status IN ?",
		eventID, userID, []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted}).
		First(&participant).Error; err != nil {
		return c.Status(403).JSON(RatingResponse{
			Success: false,
			Message: "Вы не участвовали в этом ивенте",
		})
	}

	var req SubmitRatingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация рейтингов
	if err := rc.validateRatingsRequest(&req, uint(eventID), userID); err != nil {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Начинаем транзакцию
	tx := rc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Обрабатываем каждый рейтинг
	for _, ratingReq := range req.Ratings {
		// Проверяем, не оценивал ли уже пользователь этого участника
		var existingRating models.Rating
		if err := tx.Where("event_id = ? AND from_user_id = ? AND to_user_id = ?",
			eventID, userID, ratingReq.TargetUserID).First(&existingRating).Error; err == nil {
			tx.Rollback()
			return c.Status(409).JSON(RatingResponse{
				Success: false,
				Message: "Вы уже оценили этого участника",
			})
		}

		// Создаем рейтинг
		rating := models.Rating{
			EventID:    uint(eventID),
			FromUserID: userID,
			ToUserID:   ratingReq.TargetUserID,
			Score:      ratingReq.Score,
			Comment:    ratingReq.Comment,
		}

		if err := tx.Create(&rating).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(RatingResponse{
				Success: false,
				Message: "Ошибка при сохранении рейтинга",
			})
		}

		// Обновляем сводку рейтингов целевого пользователя
		if err := rc.updateUserRatingSummary(tx, ratingReq.TargetUserID); err != nil {
			tx.Rollback()
			return c.Status(500).JSON(RatingResponse{
				Success: false,
				Message: "Ошибка при обновлении рейтинга пользователя",
			})
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(RatingResponse{
			Success: false,
			Message: "Ошибка при сохранении рейтингов",
		})
	}

	return c.JSON(RatingResponse{
		Success: true,
		Message: "Рейтинги успешно отправлены",
	})
}

// UploadPhotos загружает фотографии после завершения ивента
func (rc *RatingController) UploadPhotos(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := rc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(RatingResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента
	var event models.Event
	if err := rc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(RatingResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	// Проверяем, что ивент завершен
	if event.IsActive {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Нельзя загружать фотографии незавершенного ивента",
		})
	}

	// Проверяем, что пользователь участвовал в ивенте
	var participant models.EventParticipant
	if err := rc.DB.Where("event_id = ? AND user_id = ? AND status IN ?",
		eventID, userID, []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted}).
		First(&participant).Error; err != nil {
		return c.Status(403).JSON(RatingResponse{
			Success: false,
			Message: "Вы не участвовали в этом ивенте",
		})
	}

	// Получаем загруженные файлы
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Ошибка при обработке файлов",
		})
	}

	files := form.File["photos"]
	if len(files) == 0 {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Необходимо загрузить хотя бы одну фотографию",
		})
	}

	// Ограничиваем количество фотографий
	if len(files) > 10 {
		return c.Status(400).JSON(RatingResponse{
			Success: false,
			Message: "Можно загрузить не более 10 фотографий",
		})
	}

	// Начинаем транзакцию
	tx := rc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Сохраняем каждую фотографию
	for _, file := range files {
		// Валидация файла
		if err := rc.validatePhotoFile(file); err != nil {
			tx.Rollback()
			return c.Status(400).JSON(RatingResponse{
				Success: false,
				Message: err.Error(),
			})
		}

		// Генерируем уникальное имя файла
		fileName := rc.generateFileName(uint(eventID), userID, file.Filename)

		// Сохраняем файл
		filePath := "uploads/events/" + fileName
		if err := c.SaveFile(file, filePath); err != nil {
			tx.Rollback()
			return c.Status(500).JSON(RatingResponse{
				Success: false,
				Message: "Ошибка при сохранении файла",
			})
		}

		// Создаем запись в базе данных
		eventPhoto := models.EventPhotoPost{
			EventID:    uint(eventID),
			UploaderID: userID,
			FilePath:   filePath,
		}

		if err := tx.Create(&eventPhoto).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(RatingResponse{
				Success: false,
				Message: "Ошибка при сохранении информации о фотографии",
			})
		}
	}

	// Подтверждаем транзакцию
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(RatingResponse{
			Success: false,
			Message: "Ошибка при сохранении фотографий",
		})
	}

	return c.JSON(RatingResponse{
		Success: true,
		Message: "Фотографии успешно загружены",
	})
}

// GetUserRatings получает рейтинг пользователя
func (rc *RatingController) GetUserRatings(c *fiber.Ctx) error {
	// Получаем ID пользователя
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(UserRatingResponse{
			Success: false,
			Message: "Неверный ID пользователя",
		})
	}

	// Получаем сводку рейтингов пользователя
	var ratingSummary models.UserRatingSummary
	if err := rc.DB.Where("user_id = ?", userID).First(&ratingSummary).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Если сводки нет, создаем пустую
			ratingSummary = models.UserRatingSummary{
				UserID:       uint(userID),
				TotalScore:   0,
				VotesCount:   0,
				AverageScore: 0.0,
			}
		} else {
			return c.Status(500).JSON(UserRatingResponse{
				Success: false,
				Message: "Ошибка при получении рейтинга пользователя",
			})
		}
	}

	return c.JSON(UserRatingResponse{
		Success:       true,
		Message:       "Рейтинг пользователя получен",
		RatingSummary: &ratingSummary,
	})
}

// GetEventRatings получает рейтинги по ивенту
func (rc *RatingController) GetEventRatings(c *fiber.Ctx) error {
	// Получаем пользователя из JWT токена
	userID, err := rc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(RatingsResponse{
			Success: false,
			Message: "Неавторизованный доступ",
		})
	}

	// Получаем ID ивента
	eventID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(RatingsResponse{
			Success: false,
			Message: "Неверный ID ивента",
		})
	}

	// Проверяем существование ивента
	var event models.Event
	if err := rc.DB.First(&event, eventID).Error; err != nil {
		return c.Status(404).JSON(RatingsResponse{
			Success: false,
			Message: "Ивент не найден",
		})
	}

	// Проверяем права доступа (только участники и создатель могут видеть рейтинги)
	var participant models.EventParticipant
	if err := rc.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&participant).Error; err != nil {
		if event.CreatorID != userID {
			return c.Status(403).JSON(RatingsResponse{
				Success: false,
				Message: "Нет прав для просмотра рейтингов",
			})
		}
	}

	// Получаем рейтинги
	var ratings []models.Rating
	if err := rc.DB.Where("event_id = ?", eventID).
		Preload("FromUser").
		Preload("ToUser").
		Order("created_at DESC").
		Find(&ratings).Error; err != nil {
		return c.Status(500).JSON(RatingsResponse{
			Success: false,
			Message: "Ошибка при получении рейтингов",
		})
	}

	// Получаем общее количество
	var total int64
	if err := rc.DB.Model(&models.Rating{}).Where("event_id = ?", eventID).Count(&total).Error; err != nil {
		return c.Status(500).JSON(RatingsResponse{
			Success: false,
			Message: "Ошибка при получении количества рейтингов",
		})
	}

	return c.JSON(RatingsResponse{
		Success: true,
		Message: "Рейтинги получены",
		Ratings: ratings,
		Total:   total,
	})
}

// Вспомогательные методы

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (rc *RatingController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
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

// validateRatingsRequest валидирует запрос рейтингов
func (rc *RatingController) validateRatingsRequest(req *SubmitRatingsRequest, eventID, fromUserID uint) error {
	if len(req.Ratings) == 0 {
		return fiber.NewError(400, "Необходимо указать хотя бы один рейтинг")
	}

	// Получаем список участников ивента
	var participants []models.EventParticipant
	if err := rc.DB.Where("event_id = ? AND status IN ?",
		eventID, []string{models.ParticipantStatusJoined, models.ParticipantStatusAccepted}).
		Find(&participants).Error; err != nil {
		return fiber.NewError(500, "Ошибка при получении списка участников")
	}

	// Создаем карту участников для быстрой проверки
	participantMap := make(map[uint]bool)
	for _, p := range participants {
		participantMap[p.UserID] = true
	}

	// Проверяем каждый рейтинг
	ratedUsers := make(map[uint]bool)
	for _, rating := range req.Ratings {
		// Проверяем, что пользователь не оценивает сам себя
		if rating.TargetUserID == fromUserID {
			return fiber.NewError(400, "Нельзя оценивать самого себя")
		}

		// Проверяем, что целевой пользователь участвовал в ивенте
		if !participantMap[rating.TargetUserID] {
			return fiber.NewError(400, "Пользователь с ID "+strconv.Itoa(int(rating.TargetUserID))+" не участвовал в ивенте")
		}

		// Проверяем диапазон оценки
		if rating.Score < 1 || rating.Score > 10 {
			return fiber.NewError(400, "Оценка должна быть от 1 до 10")
		}

		// Проверяем длину комментария
		if len(rating.Comment) > 500 {
			return fiber.NewError(400, "Комментарий не должен превышать 500 символов")
		}

		// Проверяем, что пользователь не оценивается дважды в одном запросе
		if ratedUsers[rating.TargetUserID] {
			return fiber.NewError(400, "Нельзя оценить одного пользователя дважды")
		}
		ratedUsers[rating.TargetUserID] = true
	}

	return nil
}

// updateUserRatingSummary обновляет сводку рейтингов пользователя
func (rc *RatingController) updateUserRatingSummary(tx *gorm.DB, userID uint) error {
	// Получаем все рейтинги пользователя
	var ratings []models.Rating
	if err := tx.Where("to_user_id = ?", userID).Find(&ratings).Error; err != nil {
		return err
	}

	// Вычисляем общую сумму и количество
	totalScore := 0
	votesCount := len(ratings)

	for _, rating := range ratings {
		totalScore += rating.Score
	}

	// Вычисляем средний рейтинг
	averageScore := 0.0
	if votesCount > 0 {
		averageScore = float64(totalScore) / float64(votesCount)
	}

	// Обновляем или создаем сводку
	var summary models.UserRatingSummary
	if err := tx.Where("user_id = ?", userID).First(&summary).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Создаем новую сводку
			summary = models.UserRatingSummary{
				UserID:       userID,
				TotalScore:   totalScore,
				VotesCount:   votesCount,
				AverageScore: averageScore,
			}
			return tx.Create(&summary).Error
		}
		return err
	}

	// Обновляем существующую сводку
	summary.TotalScore = totalScore
	summary.VotesCount = votesCount
	summary.AverageScore = averageScore
	return tx.Save(&summary).Error
}

// validatePhotoFile валидирует загружаемый файл фотографии
func (rc *RatingController) validatePhotoFile(file *multipart.FileHeader) error {
	// Проверяем размер файла (максимум 10MB)
	if file.Size > 10*1024*1024 {
		return fiber.NewError(400, "Размер файла не должен превышать 10MB")
	}

	// Проверяем расширение файла
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	fileName := strings.ToLower(file.Filename)

	validExtension := false
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(fileName, ext) {
			validExtension = true
			break
		}
	}

	if !validExtension {
		return fiber.NewError(400, "Неподдерживаемый формат файла. Разрешены: JPG, JPEG, PNG, GIF, WEBP")
	}

	return nil
}

// generateFileName генерирует уникальное имя файла
func (rc *RatingController) generateFileName(eventID, userID uint, originalName string) string {
	// Используем timestamp + eventID + userID + оригинальное имя
	timestamp := strconv.FormatInt(rc.getCurrentTimestamp(), 10)
	return timestamp + "_" + strconv.Itoa(int(eventID)) + "_" + strconv.Itoa(int(userID)) + "_" + originalName
}

// getCurrentTimestamp возвращает текущий timestamp
func (rc *RatingController) getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
