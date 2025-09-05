package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"toloko-backend/controllers"
	"toloko-backend/models"
	"toloko-backend/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupRatingTestDB создает тестовую базу данных в памяти для тестов рейтингов
func setupRatingTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}

	// Автомиграция
	db.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.Inventory{},
		&models.EventInventory{},
		&models.EventPhoto{},
		&models.EventParticipant{},
		&models.ParticipantInventory{},
		&models.EventPhotoPost{},
		&models.Rating{},
		&models.UserRatingSummary{},
		&models.Complaint{},
	)

	// Создаем тестовых пользователей
	users := []models.User{
		{
			Name:         "Test User 1",
			Email:        "test1@example.com",
			PasswordHash: "hashed_password_1",
			IsActive:     true,
		},
		{
			Name:         "Test User 2",
			Email:        "test2@example.com",
			PasswordHash: "hashed_password_2",
			IsActive:     true,
		},
		{
			Name:         "Test User 3",
			Email:        "test3@example.com",
			PasswordHash: "hashed_password_3",
			IsActive:     true,
		},
	}
	for _, user := range users {
		db.Create(&user)
	}

	// Создаем тестовый ивент (завершенный)
	event := models.Event{
		CreatorID:       1,
		Title:           "Test Event",
		Description:     "Test Description",
		Latitude:        55.7558,
		Longitude:       37.6176,
		StartTime:       time.Now().Add(-2 * time.Hour),
		EndTime:         time.Now().Add(-1 * time.Hour),
		JoinMode:        "free",
		MinParticipants: 1,
		MaxParticipants: 10,
		IsActive:        false, // Завершенный ивент
	}
	db.Create(&event)

	// Создаем участников
	participants := []models.EventParticipant{
		{
			EventID: 1,
			UserID:  2,
			Status:  "joined",
		},
		{
			EventID: 1,
			UserID:  3,
			Status:  "joined",
		},
	}
	for _, participant := range participants {
		db.Create(&participant)
	}

	return db
}

// createRatingTestApp создает тестовое приложение Fiber для рейтингов
func createRatingTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New()

	// Создаем контроллеры
	ratingController := controllers.NewRatingController(db)

	// Настраиваем маршруты
	routes.SetupRatingRoutes(app, ratingController)

	return app
}

func TestSubmitRatings(t *testing.T) {
	db := setupRatingTestDB()
	app := createRatingTestApp(db)

	// Тест успешной отправки рейтингов
	t.Run("Successfully submit ratings", func(t *testing.T) {
		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 2,
					"score":          8,
					"comment":        "Отличная работа!",
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/1/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что рейтинг создан в базе данных
		var rating models.Rating
		err = db.Where("event_id = ? AND from_user_id = ? AND to_user_id = ?", 1, 1, 2).First(&rating).Error
		assert.NoError(t, err)
		assert.Equal(t, 8, rating.Score)
		assert.Equal(t, "Отличная работа!", rating.Comment)
	})

	// Тест попытки оценить самого себя
	t.Run("Rate yourself", func(t *testing.T) {
		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 2,
					"score":          5,
					"comment":        "Тест",
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/1/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест попытки оценить участника, который не участвовал в ивенте
	t.Run("Rate non-participant", func(t *testing.T) {
		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 999,
					"score":          5,
					"comment":        "Тест",
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/1/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест попытки оценить участников незавершенного ивента
	t.Run("Rate participants of active event", func(t *testing.T) {
		// Создаем активный ивент
		activeEvent := models.Event{
			CreatorID:       1,
			Title:           "Active Event",
			Description:     "Active Description",
			Latitude:        55.7558,
			Longitude:       37.6176,
			StartTime:       time.Now().Add(1 * time.Hour),
			EndTime:         time.Now().Add(3 * time.Hour),
			JoinMode:        "free",
			MinParticipants: 1,
			MaxParticipants: 10,
			IsActive:        true,
		}
		db.Create(&activeEvent)

		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 2,
					"score":          5,
					"comment":        "Тест",
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/2/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestGetEventRatings(t *testing.T) {
	db := setupRatingTestDB()
	app := createRatingTestApp(db)

	// Создаем тестовые рейтинги
	ratings := []models.Rating{
		{
			EventID:    1,
			FromUserID: 2,
			ToUserID:   3,
			Score:      8,
			Comment:    "Отличная работа!",
		},
		{
			EventID:    1,
			FromUserID: 3,
			ToUserID:   2,
			Score:      7,
			Comment:    "Хорошая работа!",
		},
	}
	for _, rating := range ratings {
		db.Create(&rating)
	}

	// Тест получения рейтингов по ивенту
	t.Run("Get event ratings", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/events/1/ratings", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestGetUserRating(t *testing.T) {
	db := setupRatingTestDB()
	app := createRatingTestApp(db)

	// Создаем сводку рейтингов пользователя
	ratingSummary := models.UserRatingSummary{
		UserID:       2,
		TotalScore:   15,
		VotesCount:   2,
		AverageScore: 7.5,
	}
	db.Create(&ratingSummary)

	// Тест получения рейтинга пользователя
	t.Run("Get user rating", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/2/ratings", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// Тест получения рейтинга несуществующего пользователя
	t.Run("Get non-existent user rating", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/999/ratings", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode) // Должен вернуть пустую сводку
	})
}

func TestUploadPhotos(t *testing.T) {
	db := setupRatingTestDB()
	app := createRatingTestApp(db)

	// Тест загрузки фотографий
	t.Run("Upload photos", func(t *testing.T) {
		// Создаем multipart запрос с тестовыми файлами
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Добавляем тестовый файл
		part, _ := writer.CreateFormFile("photos", "test.jpg")
		part.Write([]byte("fake image content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/events/1/photos", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// Тест загрузки фотографий без файлов
	t.Run("Upload photos without files", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest("POST", "/events/1/photos", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestRatingValidation(t *testing.T) {
	db := setupRatingTestDB()
	app := createRatingTestApp(db)

	// Тест валидации оценки (должна быть от 1 до 10)
	t.Run("Invalid score validation", func(t *testing.T) {
		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 2,
					"score":          15, // Неверная оценка
					"comment":        "Тест",
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/1/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест валидации комментария (максимум 500 символов)
	t.Run("Long comment validation", func(t *testing.T) {
		longComment := string(make([]byte, 501)) // 501 символ
		for i := range longComment {
			longComment = longComment[:i] + "a" + longComment[i+1:]
		}

		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 2,
					"score":          5,
					"comment":        longComment,
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/1/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestDuplicateRating(t *testing.T) {
	db := setupRatingTestDB()
	app := createRatingTestApp(db)

	// Создаем существующий рейтинг
	existingRating := models.Rating{
		EventID:    1,
		FromUserID: 2,
		ToUserID:   3,
		Score:      5,
		Comment:    "Старый комментарий",
	}
	db.Create(&existingRating)

	// Тест попытки повторного оценивания
	t.Run("Duplicate rating", func(t *testing.T) {
		ratingsData := map[string]interface{}{
			"ratings": []map[string]interface{}{
				{
					"target_user_id": 2,
					"score":          8,
					"comment":        "Новый комментарий",
				},
			},
		}

		jsonData, _ := json.Marshal(ratingsData)
		req := httptest.NewRequest("POST", "/events/1/ratings", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.StatusCode)
	})
}
