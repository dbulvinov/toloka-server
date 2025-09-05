package main

import (
	"bytes"
	"encoding/json"
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

// setupComplaintTestDB создает тестовую базу данных в памяти для тестов жалоб
func setupComplaintTestDB() *gorm.DB {
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

	// Создаем тестовый ивент
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
		IsActive:        false,
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

// createComplaintTestApp создает тестовое приложение Fiber для жалоб
func createComplaintTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New()

	// Создаем контроллеры
	complaintController := controllers.NewComplaintController(db)

	// Настраиваем маршруты
	routes.SetupComplaintRoutes(app, complaintController)

	return app
}

func TestSubmitComplaint(t *testing.T) {
	db := setupComplaintTestDB()
	app := createComplaintTestApp(db)

	// Тест успешной подачи жалобы
	t.Run("Successfully submit complaint", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "inappropriate_behavior",
			"reason_text":   "Пользователь вел себя неподобающим образом во время ивента",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		// Проверяем, что жалоба создана в базе данных
		var complaint models.Complaint
		err = db.Where("event_id = ? AND from_user_id = ? AND about_user_id = ?", 1, 1, 2).First(&complaint).Error
		assert.NoError(t, err)
		assert.Equal(t, "inappropriate_behavior", complaint.ReasonCode)
		assert.Equal(t, "open", complaint.Status)
	})

	// Тест попытки подать жалобу на самого себя
	t.Run("Complain about yourself", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "inappropriate_behavior",
			"reason_text":   "Тест",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест попытки подать жалобу на пользователя, который не участвовал в ивенте
	t.Run("Complain about non-participant", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 999,
			"reason_code":   "inappropriate_behavior",
			"reason_text":   "Тест",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест попытки подать жалобу без авторизации
	t.Run("Submit complaint without authorization", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "inappropriate_behavior",
			"reason_text":   "Тест",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	// Тест попытки подать жалобу с неверным кодом причины
	t.Run("Submit complaint with invalid reason code", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "invalid_reason",
			"reason_text":   "Тест",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест попытки подать жалобу с коротким описанием
	t.Run("Submit complaint with short description", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "inappropriate_behavior",
			"reason_text":   "Коротко",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	// Тест попытки подать жалобу с длинным описанием
	t.Run("Submit complaint with long description", func(t *testing.T) {
		longText := string(make([]byte, 1001)) // 1001 символ
		for i := range longText {
			longText = longText[:i] + "a" + longText[i+1:]
		}

		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "inappropriate_behavior",
			"reason_text":   longText,
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestGetComplaints(t *testing.T) {
	db := setupComplaintTestDB()
	app := createComplaintTestApp(db)

	// Создаем тестовые жалобы
	complaints := []models.Complaint{
		{
			EventID:     1,
			FromUserID:  2,
			AboutUserID: 3,
			ReasonCode:  "inappropriate_behavior",
			ReasonText:  "Неподобающее поведение",
			Status:      "open",
		},
		{
			EventID:     1,
			FromUserID:  3,
			AboutUserID: 2,
			ReasonCode:  "no_show",
			ReasonText:  "Не явился на мероприятие",
			Status:      "resolved",
		},
	}
	for _, complaint := range complaints {
		db.Create(&complaint)
	}

	// Тест получения списка жалоб (только для модераторов)
	t.Run("Get complaints", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/complaints", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// Тест получения жалоб с фильтрацией по статусу
	t.Run("Get complaints with status filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/complaints?status=open", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// Тест получения жалоб с фильтрацией по ивенту
	t.Run("Get complaints with event filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/complaints?event_id=1", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestUpdateComplaintStatus(t *testing.T) {
	db := setupComplaintTestDB()
	app := createComplaintTestApp(db)

	// Создаем тестовую жалобу
	complaint := models.Complaint{
		EventID:     1,
		FromUserID:  2,
		AboutUserID: 3,
		ReasonCode:  "inappropriate_behavior",
		ReasonText:  "Неподобающее поведение",
		Status:      "open",
	}
	db.Create(&complaint)

	// Тест обновления статуса жалобы
	t.Run("Update complaint status", func(t *testing.T) {
		statusData := map[string]interface{}{
			"status": "under_review",
		}

		jsonData, _ := json.Marshal(statusData)
		req := httptest.NewRequest("PUT", "/complaints/1/status", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что статус обновился
		var updatedComplaint models.Complaint
		err = db.Where("id = ?", 1).First(&updatedComplaint).Error
		assert.NoError(t, err)
		assert.Equal(t, "under_review", updatedComplaint.Status)
	})

	// Тест обновления статуса несуществующей жалобы
	t.Run("Update non-existent complaint status", func(t *testing.T) {
		statusData := map[string]interface{}{
			"status": "resolved",
		}

		jsonData, _ := json.Marshal(statusData)
		req := httptest.NewRequest("PUT", "/complaints/999/status", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})

	// Тест обновления статуса с неверным статусом
	t.Run("Update complaint status with invalid status", func(t *testing.T) {
		statusData := map[string]interface{}{
			"status": "invalid_status",
		}

		jsonData, _ := json.Marshal(statusData)
		req := httptest.NewRequest("PUT", "/complaints/1/status", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestGetComplaintReasons(t *testing.T) {
	db := setupComplaintTestDB()
	app := createComplaintTestApp(db)

	// Тест получения списка причин жалоб
	t.Run("Get complaint reasons", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/complaints/reasons", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestGetUserComplaints(t *testing.T) {
	db := setupComplaintTestDB()
	app := createComplaintTestApp(db)

	// Создаем тестовые жалобы пользователя
	complaints := []models.Complaint{
		{
			EventID:     1,
			FromUserID:  2,
			AboutUserID: 3,
			ReasonCode:  "inappropriate_behavior",
			ReasonText:  "Неподобающее поведение",
			Status:      "open",
		},
		{
			EventID:     1,
			FromUserID:  2,
			AboutUserID: 3,
			ReasonCode:  "no_show",
			ReasonText:  "Не явился на мероприятие",
			Status:      "resolved",
		},
	}
	for _, complaint := range complaints {
		db.Create(&complaint)
	}

	// Тест получения жалоб пользователя
	t.Run("Get user complaints", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/complaints/my", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	// Тест получения жалоб пользователя с фильтрацией по статусу
	t.Run("Get user complaints with status filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/complaints/my?status=open", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestDuplicateComplaint(t *testing.T) {
	db := setupComplaintTestDB()
	app := createComplaintTestApp(db)

	// Создаем существующую жалобу
	existingComplaint := models.Complaint{
		EventID:     1,
		FromUserID:  2,
		AboutUserID: 3,
		ReasonCode:  "inappropriate_behavior",
		ReasonText:  "Существующая жалоба",
		Status:      "open",
	}
	db.Create(&existingComplaint)

	// Тест попытки подать жалобу на того же пользователя повторно
	t.Run("Duplicate complaint", func(t *testing.T) {
		complaintData := map[string]interface{}{
			"about_user_id": 2,
			"reason_code":   "no_show",
			"reason_text":   "Новая жалоба на того же пользователя",
		}

		jsonData, _ := json.Marshal(complaintData)
		req := httptest.NewRequest("POST", "/events/1/complaints", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.StatusCode)
	})
}
