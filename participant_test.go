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

// setupParticipantTestDB создает тестовую базу данных в памяти для тестов участников
func setupParticipantTestDB() *gorm.DB {
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

	// Создаем тестовый инвентарь
	inventory := models.Inventory{
		Name:      "Test Inventory",
		IsCustom:  false,
		CreatedBy: 0,
		IsActive:  true,
	}
	db.Create(&inventory)

	// Создаем тестовый ивент
	event := models.Event{
		CreatorID:       1,
		Title:           "Test Event",
		Description:     "Test Description",
		Latitude:        55.7558,
		Longitude:       37.6176,
		StartTime:       time.Now().Add(24 * time.Hour),
		EndTime:         time.Now().Add(26 * time.Hour),
		JoinMode:        "free",
		MinParticipants: 1,
		MaxParticipants: 10,
		IsActive:        true,
	}
	db.Create(&event)

	// Создаем участника для тестов
	participant := models.EventParticipant{
		EventID:  1,
		UserID:   2,
		Status:   "joined",
		JoinedAt: &time.Time{},
	}
	now := time.Now()
	participant.JoinedAt = &now
	db.Create(&participant)

	return db
}

// createTestApp создает тестовое приложение Fiber
func createParticipantTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New()

	// Создаем контроллеры
	participantController := controllers.NewParticipantController(db)

	// Настраиваем маршруты
	routes.SetupParticipantRoutes(app, participantController)

	return app
}

func TestJoinEvent(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Тест успешного вступления в ивент
	t.Run("Successfully join event", func(t *testing.T) {
		joinData := map[string]interface{}{
			"desired_inventory": []map[string]interface{}{
				{
					"inventory_item_id": 1,
					"custom_name":       "",
					"quantity":          2,
				},
			},
		}

		jsonData, _ := json.Marshal(joinData)
		req := httptest.NewRequest("POST", "/events/1/join", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		// Проверяем, что участник создан в базе данных
		var participant models.EventParticipant
		err = db.Where("event_id = ? AND user_id = ?", 1, 1).First(&participant).Error
		assert.NoError(t, err)
		assert.Equal(t, "joined", participant.Status)
	})

	// Тест попытки вступления в несуществующий ивент
	t.Run("Join non-existent event", func(t *testing.T) {
		joinData := map[string]interface{}{
			"desired_inventory": []map[string]interface{}{},
		}

		jsonData, _ := json.Marshal(joinData)
		req := httptest.NewRequest("POST", "/events/999/join", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})

	// Тест попытки вступления без авторизации
	t.Run("Join without authorization", func(t *testing.T) {
		joinData := map[string]interface{}{
			"desired_inventory": []map[string]interface{}{},
		}

		jsonData, _ := json.Marshal(joinData)
		req := httptest.NewRequest("POST", "/events/1/join", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

func TestLeaveEvent(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Создаем участника
	participant := models.EventParticipant{
		EventID: 1,
		UserID:  2,
		Status:  "joined",
	}
	db.Create(&participant)

	// Тест успешного выхода из ивента
	t.Run("Successfully leave event", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/events/1/leave", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что статус участника изменился
		var updatedParticipant models.EventParticipant
		err = db.Where("event_id = ? AND user_id = ?", 1, 2).First(&updatedParticipant).Error
		assert.NoError(t, err)
		assert.Equal(t, "left", updatedParticipant.Status)
		assert.NotNil(t, updatedParticipant.LeftAt)
	})

	// Тест попытки выхода из ивента, в котором пользователь не участвует
	t.Run("Leave event not participated", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/events/1/leave", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})
}

func TestGetParticipants(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

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

	// Тест получения списка участников
	t.Run("Get participants", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/events/1/participants", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestGetApplications(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Создаем заявки
	applications := []models.EventParticipant{
		{
			EventID: 1,
			UserID:  2,
			Status:  "pending",
		},
		{
			EventID: 1,
			UserID:  3,
			Status:  "pending",
		},
	}
	for _, application := range applications {
		db.Create(&application)
	}

	// Тест получения списка заявок
	t.Run("Get applications", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/events/1/applications", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestApproveApplication(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Создаем заявку
	application := models.EventParticipant{
		EventID: 1,
		UserID:  2,
		Status:  "pending",
	}
	db.Create(&application)

	// Тест одобрения заявки
	t.Run("Approve application", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/events/1/applications/1/approve", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что статус заявки изменился
		var updatedApplication models.EventParticipant
		err = db.Where("id = ?", 1).First(&updatedApplication).Error
		assert.NoError(t, err)
		assert.Equal(t, "accepted", updatedApplication.Status)
	})
}

func TestRejectApplication(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Создаем заявку
	application := models.EventParticipant{
		EventID: 1,
		UserID:  2,
		Status:  "pending",
	}
	db.Create(&application)

	// Тест отклонения заявки
	t.Run("Reject application", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/events/1/applications/1/reject", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что статус заявки изменился
		var updatedApplication models.EventParticipant
		err = db.Where("id = ?", 1).First(&updatedApplication).Error
		assert.NoError(t, err)
		assert.Equal(t, "rejected", updatedApplication.Status)
	})
}

func TestCompleteEvent(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Тест завершения ивента
	t.Run("Complete event", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/events/1/complete", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что ивент деактивирован
		var event models.Event
		err = db.Where("id = ?", 1).First(&event).Error
		assert.NoError(t, err)
		assert.False(t, event.IsActive)
	})
}

func TestUpdateParticipantInventory(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Создаем участника
	participant := models.EventParticipant{
		EventID: 1,
		UserID:  2,
		Status:  "joined",
	}
	db.Create(&participant)

	// Тест обновления инвентаря участника
	t.Run("Update participant inventory", func(t *testing.T) {
		inventoryData := map[string]interface{}{
			"inventory": []map[string]interface{}{
				{
					"inventory_item_id": 1,
					"custom_name":       "",
					"quantity":          3,
				},
			},
		}

		jsonData, _ := json.Marshal(inventoryData)
		req := httptest.NewRequest("POST", "/participants/1/inventory", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestGetInventorySummary(t *testing.T) {
	db := setupParticipantTestDB()
	app := createParticipantTestApp(db)

	// Создаем участника с инвентарем
	participant := models.EventParticipant{
		EventID: 1,
		UserID:  2,
		Status:  "joined",
	}
	db.Create(&participant)

	participantInventory := models.ParticipantInventory{
		ParticipantID:   1,
		InventoryItemID: &[]uint{1}[0],
		CustomName:      "",
		Quantity:        2,
		Brought:         false,
	}
	db.Create(&participantInventory)

	// Тест получения сводки по инвентарю
	t.Run("Get inventory summary", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/events/1/inventory-summary", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestJWT(1))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
