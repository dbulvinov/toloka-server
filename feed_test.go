package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"toloko-backend/controllers"
	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupFeedTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Event{}, &models.Subscription{})
	return db
}

func createFeedTestUser(db *gorm.DB, name, email string) *models.User {
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	db.Create(user)
	return user
}

func createFeedTestEvent(db *gorm.DB, creatorID uint, title string, startTime, endTime time.Time) *models.Event {
	event := &models.Event{
		CreatorID:       creatorID,
		Title:           title,
		Description:     "Test event description",
		Latitude:        55.7558,
		Longitude:       37.6176,
		StartTime:       startTime,
		EndTime:         endTime,
		JoinMode:        "free",
		MinParticipants: 1,
		MaxParticipants: 10,
		IsActive:        true,
	}
	db.Create(event)
	return event
}

func createFeedTestToken(userID uint, email string) string {
	token, _ := utils.GenerateJWT(userID, email)
	return token
}

func TestFeedController_GetFeed(t *testing.T) {
	db := setupFeedTestDB()
	controller := controllers.NewFeedController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createFeedTestUser(db, "User 1", "user1@test.com")
	user2 := createFeedTestUser(db, "User 2", "user2@test.com")
	user3 := createFeedTestUser(db, "User 3", "user3@test.com")

	// Создаем токен для user1
	token := createFeedTestToken(user1.ID, user1.Email)

	// Создаем подписку (user1 подписывается на user2)
	subscription := models.Subscription{
		SubscriberID:   user1.ID,
		SubscribedToID: user2.ID,
	}
	db.Create(&subscription)

	// Создаем события
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	// Событие от user2 (должно быть в ленте)
	event1 := createFeedTestEvent(db, user2.ID, "Event from User 2", futureTime, futureTime.Add(2*time.Hour))

	// Событие от user3 (не должно быть в ленте)
	createFeedTestEvent(db, user3.ID, "Event from User 3", futureTime, futureTime.Add(2*time.Hour))

	// Событие от user2 в прошлом (не должно быть в ленте)
	createFeedTestEvent(db, user2.ID, "Past Event from User 2", pastTime, pastTime.Add(2*time.Hour))

	// Настраиваем маршрут
	app.Get("/feed", controller.GetFeed)

	t.Run("Получение ленты с событиями", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		events := data["events"].([]interface{})

		// Должно быть только одно событие (от user2 в будущем)
		assert.Len(t, events, 1)

		event := events[0].(map[string]interface{})
		assert.Equal(t, event1.ID, uint(event["id"].(float64)))
	})

	t.Run("Лента пуста - нет подписок", func(t *testing.T) {
		// Создаем нового пользователя без подписок
		user4 := createFeedTestUser(db, "User 4", "user4@test.com")
		token4 := createFeedTestToken(user4.ID, user4.Email)

		req := httptest.NewRequest("GET", "/feed", nil)
		req.Header.Set("Authorization", "Bearer "+token4)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		events := data["events"].([]interface{})
		assert.Len(t, events, 0)
	})
}

func TestFeedController_GetFeedWithFilters(t *testing.T) {
	db := setupFeedTestDB()
	controller := controllers.NewFeedController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createFeedTestUser(db, "User 1", "user1@test.com")
	user2 := createFeedTestUser(db, "User 2", "user2@test.com")

	// Создаем токен для user1
	token := createFeedTestToken(user1.ID, user1.Email)

	// Создаем подписку
	subscription := models.Subscription{
		SubscriberID:   user1.ID,
		SubscribedToID: user2.ID,
	}
	db.Create(&subscription)

	// Создаем события
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	// Будущее событие
	event1 := createFeedTestEvent(db, user2.ID, "Future Event", futureTime, futureTime.Add(2*time.Hour))

	// Прошедшее событие
	event2 := createFeedTestEvent(db, user2.ID, "Past Event", pastTime, pastTime.Add(2*time.Hour))

	// Настраиваем маршрут
	app.Get("/feed/filtered", controller.GetFeedWithFilters)

	t.Run("Фильтр по статусу - upcoming", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed/filtered?status=upcoming", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		events := data["events"].([]interface{})
		assert.Len(t, events, 1)

		event := events[0].(map[string]interface{})
		assert.Equal(t, event1.ID, uint(event["id"].(float64)))
	})

	t.Run("Фильтр по статусу - past", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed/filtered?status=past", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		events := data["events"].([]interface{})
		assert.Len(t, events, 1)

		event := events[0].(map[string]interface{})
		assert.Equal(t, event2.ID, uint(event["id"].(float64)))
	})

	t.Run("Поиск по названию", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed/filtered?search=Future", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		events := data["events"].([]interface{})
		assert.Len(t, events, 1)

		event := events[0].(map[string]interface{})
		assert.Equal(t, event1.ID, uint(event["id"].(float64)))
	})
}

func TestFeedController_GetRecommendedEvents(t *testing.T) {
	db := setupFeedTestDB()
	controller := controllers.NewFeedController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createFeedTestUser(db, "User 1", "user1@test.com")
	user2 := createFeedTestUser(db, "User 2", "user2@test.com")
	user3 := createFeedTestUser(db, "User 3", "user3@test.com")

	// Создаем токен для user1
	token := createFeedTestToken(user1.ID, user1.Email)

	// Создаем подписку (user1 подписывается на user2)
	subscription := models.Subscription{
		SubscriberID:   user1.ID,
		SubscribedToID: user2.ID,
	}
	db.Create(&subscription)

	// Создаем события
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)

	// Событие от user2 (не должно быть в рекомендациях, так как user1 подписан)
	createFeedTestEvent(db, user2.ID, "Event from User 2", futureTime, futureTime.Add(2*time.Hour))

	// Событие от user3 (должно быть в рекомендациях)
	event2 := createFeedTestEvent(db, user3.ID, "Event from User 3", futureTime, futureTime.Add(2*time.Hour))

	// Событие от user1 (не должно быть в рекомендациях, так как это его собственное событие)
	createFeedTestEvent(db, user1.ID, "My Own Event", futureTime, futureTime.Add(2*time.Hour))

	// Настраиваем маршрут
	app.Get("/feed/recommended", controller.GetRecommendedEvents)

	t.Run("Получение рекомендуемых событий", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed/recommended", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		events := data["events"].([]interface{})

		// Должно быть только одно событие (от user3)
		assert.Len(t, events, 1)

		event := events[0].(map[string]interface{})
		assert.Equal(t, event2.ID, uint(event["id"].(float64)))
	})
}

func TestFeedController_GetFeedStats(t *testing.T) {
	db := setupFeedTestDB()
	controller := controllers.NewFeedController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createFeedTestUser(db, "User 1", "user1@test.com")
	user2 := createFeedTestUser(db, "User 2", "user2@test.com")

	// Создаем токен для user1
	token := createFeedTestToken(user1.ID, user1.Email)

	// Создаем подписку
	subscription := models.Subscription{
		SubscriberID:   user1.ID,
		SubscribedToID: user2.ID,
	}
	db.Create(&subscription)

	// Создаем события
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	// Будущее событие
	createFeedTestEvent(db, user2.ID, "Future Event", futureTime, futureTime.Add(2*time.Hour))

	// Прошедшее событие
	createFeedTestEvent(db, user2.ID, "Past Event", pastTime, pastTime.Add(2*time.Hour))

	// Настраиваем маршрут
	app.Get("/feed/stats", controller.GetFeedStats)

	t.Run("Получение статистики ленты", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed/stats", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["subscriptions_count"])
		assert.Equal(t, float64(1), data["upcoming_events"])
		assert.Equal(t, float64(1), data["past_events"])
	})
}
