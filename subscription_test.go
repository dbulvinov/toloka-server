package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"toloko-backend/controllers"
	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSubscriptionTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Subscription{})
	return db
}

func createTestUser(db *gorm.DB, name, email string) *models.User {
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: "hashed_password",
		IsActive:     true,
	}
	db.Create(user)
	return user
}

func createTestToken(userID uint, email string) string {
	token, _ := utils.GenerateJWT(userID, email)
	return token
}

func TestSubscriptionController_Subscribe(t *testing.T) {
	db := setupSubscriptionTestDB()
	controller := controllers.NewSubscriptionController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createTestUser(db, "User 1", "user1@test.com")
	user2 := createTestUser(db, "User 2", "user2@test.com")

	// Создаем токен для user1
	token := createTestToken(user1.ID, user1.Email)

	// Настраиваем маршрут
	app.Post("/subscriptions/:user_id", controller.Subscribe)

	t.Run("Успешная подписка", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/subscriptions/"+string(rune(user2.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		// Проверяем, что подписка создана в базе
		var subscription models.Subscription
		err = db.Where("subscriber_id = ? AND subscribed_to_id = ?", user1.ID, user2.ID).First(&subscription).Error
		assert.NoError(t, err)
	})

	t.Run("Подписка на самого себя", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/subscriptions/"+string(rune(user1.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("Подписка на несуществующего пользователя", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/subscriptions/999", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})

	t.Run("Повторная подписка", func(t *testing.T) {
		// Создаем подписку
		subscription := models.Subscription{
			SubscriberID:   user1.ID,
			SubscribedToID: user2.ID,
		}
		db.Create(&subscription)

		req := httptest.NewRequest("POST", "/subscriptions/"+string(rune(user2.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.StatusCode)
	})
}

func TestSubscriptionController_Unsubscribe(t *testing.T) {
	db := setupSubscriptionTestDB()
	controller := controllers.NewSubscriptionController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createTestUser(db, "User 1", "user1@test.com")
	user2 := createTestUser(db, "User 2", "user2@test.com")

	// Создаем токен для user1
	token := createTestToken(user1.ID, user1.Email)

	// Настраиваем маршрут
	app.Delete("/subscriptions/:user_id", controller.Unsubscribe)

	t.Run("Успешная отписка", func(t *testing.T) {
		// Создаем подписку
		subscription := models.Subscription{
			SubscriberID:   user1.ID,
			SubscribedToID: user2.ID,
		}
		db.Create(&subscription)

		req := httptest.NewRequest("DELETE", "/subscriptions/"+string(rune(user2.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Проверяем, что подписка удалена из базы
		var count int64
		db.Model(&models.Subscription{}).Where("subscriber_id = ? AND subscribed_to_id = ?", user1.ID, user2.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Отписка от несуществующей подписки", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/subscriptions/"+string(rune(user2.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})
}

func TestSubscriptionController_GetSubscriptions(t *testing.T) {
	db := setupSubscriptionTestDB()
	controller := controllers.NewSubscriptionController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createTestUser(db, "User 1", "user1@test.com")
	user2 := createTestUser(db, "User 2", "user2@test.com")
	user3 := createTestUser(db, "User 3", "user3@test.com")

	// Создаем токен для user1
	token := createTestToken(user1.ID, user1.Email)

	// Создаем подписки
	subscription1 := models.Subscription{
		SubscriberID:   user1.ID,
		SubscribedToID: user2.ID,
	}
	subscription2 := models.Subscription{
		SubscriberID:   user1.ID,
		SubscribedToID: user3.ID,
	}
	db.Create(&subscription1)
	db.Create(&subscription2)

	// Настраиваем маршрут
	app.Get("/subscriptions", controller.GetSubscriptions)

	t.Run("Получение списка подписок", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subscriptions", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		subscriptions := data["subscriptions"].([]interface{})
		assert.Len(t, subscriptions, 2)
	})
}

func TestSubscriptionController_GetSubscribers(t *testing.T) {
	db := setupSubscriptionTestDB()
	controller := controllers.NewSubscriptionController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createTestUser(db, "User 1", "user1@test.com")
	user2 := createTestUser(db, "User 2", "user2@test.com")
	user3 := createTestUser(db, "User 3", "user3@test.com")

	// Создаем токен для user1
	token := createTestToken(user1.ID, user1.Email)

	// Создаем подписки (user2 и user3 подписываются на user1)
	subscription1 := models.Subscription{
		SubscriberID:   user2.ID,
		SubscribedToID: user1.ID,
	}
	subscription2 := models.Subscription{
		SubscriberID:   user3.ID,
		SubscribedToID: user1.ID,
	}
	db.Create(&subscription1)
	db.Create(&subscription2)

	// Настраиваем маршрут
	app.Get("/subscribers", controller.GetSubscribers)

	t.Run("Получение списка подписчиков", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subscribers", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		subscribers := data["subscribers"].([]interface{})
		assert.Len(t, subscribers, 2)
	})
}

func TestSubscriptionController_CheckSubscription(t *testing.T) {
	db := setupSubscriptionTestDB()
	controller := controllers.NewSubscriptionController(db)
	app := fiber.New()

	// Создаем тестовых пользователей
	user1 := createTestUser(db, "User 1", "user1@test.com")
	user2 := createTestUser(db, "User 2", "user2@test.com")

	// Создаем токен для user1
	token := createTestToken(user1.ID, user1.Email)

	// Настраиваем маршрут
	app.Get("/subscriptions/check/:user_id", controller.CheckSubscription)

	t.Run("Проверка подписки - не подписан", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/subscriptions/check/"+string(rune(user2.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.False(t, data["is_subscribed"].(bool))
	})

	t.Run("Проверка подписки - подписан", func(t *testing.T) {
		// Создаем подписку
		subscription := models.Subscription{
			SubscriberID:   user1.ID,
			SubscribedToID: user2.ID,
		}
		db.Create(&subscription)

		req := httptest.NewRequest("GET", "/subscriptions/check/"+string(rune(user2.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.True(t, data["is_subscribed"].(bool))
	})
}
