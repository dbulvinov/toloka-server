package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"toloko-backend/controllers"
	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestCreateConversation(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	app := fiber.New()
	conversationController := controllers.NewConversationController(db)

	// Middleware для установки user_id
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Post("/conversations", conversationController.CreateConversation)

	// Тест создания диалога
	reqBody := map[string]interface{}{
		"user_b_id": user2ID,
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/conversations", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Проверяем, что диалог создан
	var conversation models.Conversation
	err = db.Where("user_a_id = ? AND user_b_id = ?", user1ID, user2ID).First(&conversation).Error
	assert.NoError(t, err)
	assert.Equal(t, user1ID, conversation.UserAID)
	assert.Equal(t, user2ID, conversation.UserBID)
}

func TestGetConversations(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	app := fiber.New()
	conversationController := controllers.NewConversationController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/conversations", conversationController.GetConversations)

	req := httptest.NewRequest("GET", "/conversations", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetConversation(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	app := fiber.New()
	conversationController := controllers.NewConversationController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/conversations/:id", conversationController.GetConversation)

	req := httptest.NewRequest("GET", "/conversations/1", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestMarkConversationAsRead(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	app := fiber.New()
	conversationController := controllers.NewConversationController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Put("/conversations/:id/read", conversationController.MarkAsRead)

	req := httptest.NewRequest("PUT", "/conversations/1/read", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDeleteConversation(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	app := fiber.New()
	conversationController := controllers.NewConversationController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Delete("/conversations/:id", conversationController.DeleteConversation)

	req := httptest.NewRequest("DELETE", "/conversations/1", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Проверяем, что диалог удален
	var count int64
	db.Model(&models.Conversation{}).Where("id = ?", conversation.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
