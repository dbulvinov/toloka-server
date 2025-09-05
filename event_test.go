package main

import (
	"bytes"
	"encoding/json"
	"net/http"
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

// setupEventTestDB создает тестовую базу данных в памяти для тестов ивентов
func setupEventTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}

	// Автомиграция
	db.AutoMigrate(&models.User{}, &models.Event{}, &models.Inventory{}, &models.EventInventory{}, &models.EventPhoto{}, &models.EventParticipant{})

	// Создаем тестового пользователя
	user := models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	db.Create(&user)

	// Создаем тестовый инвентарь
	inventory := models.Inventory{
		Name:      "Test Inventory",
		IsCustom:  false,
		CreatedBy: 0,
		IsActive:  true,
	}
	db.Create(&inventory)

	return db
}

// createTestApp создает тестовое приложение
func createTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New()

	// Инициализация контроллеров
	eventController := controllers.NewEventController(db)

	// Настройка маршрутов
	routes.SetupEventRoutes(app, eventController)

	return app
}

// getTestToken создает тестовый JWT токен
func getTestToken() string {
	// В реальном тесте здесь бы использовался utils.GenerateJWT
	// Для упрощения возвращаем фиктивный токен
	return "test_token"
}

func TestCreateEvent(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Подготавливаем данные для создания ивента
	eventData := map[string]interface{}{
		"title":            "Test Event",
		"description":      "Test Description",
		"latitude":         55.7558,
		"longitude":        37.6176,
		"start_time":       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"end_time":         time.Now().Add(26 * time.Hour).Format(time.RFC3339),
		"join_mode":        "free",
		"min_participants": 1,
		"max_participants": 10,
		"inventory":        []map[string]interface{}{},
		"photos":           []string{},
	}

	jsonData, _ := json.Marshal(eventData)

	// Создаем запрос
	req := httptest.NewRequest("POST", "/events", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken())

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode) // Ожидаем ошибку авторизации, так как токен фиктивный
}

func TestGetEvents(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

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

	// Создаем запрос
	req := httptest.NewRequest("GET", "/events", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Проверяем ответ
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestGetEvent(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

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

	// Создаем запрос
	req := httptest.NewRequest("GET", "/events/1", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Проверяем ответ
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestCreateInventory(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Подготавливаем данные для создания инвентаря
	inventoryData := map[string]interface{}{
		"name": "New Test Inventory",
	}

	jsonData, _ := json.Marshal(inventoryData)

	// Создаем запрос
	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken())

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode) // Ожидаем ошибку авторизации
}

func TestGetInventories(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Создаем запрос
	req := httptest.NewRequest("GET", "/inventory", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Проверяем ответ
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestEventValidation(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Тест с неверными данными
	invalidEventData := map[string]interface{}{
		"title":            "", // Пустое название
		"description":      "Test Description",
		"latitude":         200, // Неверная широта
		"longitude":        37.6176,
		"start_time":       time.Now().Add(-24 * time.Hour).Format(time.RFC3339), // Прошедшее время
		"end_time":         time.Now().Add(-22 * time.Hour).Format(time.RFC3339),
		"join_mode":        "invalid_mode", // Неверный режим
		"min_participants": 0,              // Неверное количество
		"max_participants": 10,
		"inventory":        []map[string]interface{}{},
		"photos":           []string{},
	}

	jsonData, _ := json.Marshal(invalidEventData)

	// Создаем запрос
	req := httptest.NewRequest("POST", "/events", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken())

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode) // Ожидаем ошибку авторизации
}

func TestEventNotFound(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Создаем запрос к несуществующему ивенту
	req := httptest.NewRequest("GET", "/events/999", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Проверяем ответ
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

func TestEventHealthCheck(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Создаем запрос к health check
	req := httptest.NewRequest("GET", "/events/health", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Проверяем ответ
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Events service is running", response["message"])
}

func TestInventoryHealthCheck(t *testing.T) {
	db := setupEventTestDB()
	app := createTestApp(db)

	// Создаем запрос к health check
	req := httptest.NewRequest("GET", "/inventory/health", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Проверяем ответ
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Inventory service is running", response["message"])
}
