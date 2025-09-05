package main

import (
	"net/http/httptest"
	"testing"

	"toloko-backend/controllers"
	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetMessages(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	// Создаем сообщения
	messages := []models.Message{
		{
			ConversationID: conversation.ID,
			FromUserID:     user1ID,
			ToUserID:       user2ID,
			Text:           "Hello",
			Status:         models.MessageStatusSent,
		},
		{
			ConversationID: conversation.ID,
			FromUserID:     user2ID,
			ToUserID:       user1ID,
			Text:           "Hi there",
			Status:         models.MessageStatusSent,
		},
	}

	for _, msg := range messages {
		db.Create(&msg)
	}

	app := fiber.New()
	messageController := controllers.NewMessageController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/conversations/:conversation_id/messages", messageController.GetMessages)

	req := httptest.NewRequest("GET", "/conversations/1/messages", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestMarkMessageAsRead(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	// Создаем сообщение
	message := models.Message{
		ConversationID: conversation.ID,
		FromUserID:     user2ID,
		ToUserID:       user1ID,
		Text:           "Hello",
		Status:         models.MessageStatusSent,
	}
	db.Create(&message)

	app := fiber.New()
	messageController := controllers.NewMessageController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Put("/messages/:id/read", messageController.MarkAsRead)

	req := httptest.NewRequest("PUT", "/messages/1/read", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Проверяем, что сообщение помечено как прочитанное
	var updatedMessage models.Message
	db.First(&updatedMessage, message.ID)
	assert.Equal(t, models.MessageStatusRead, updatedMessage.Status)
}

func TestDeleteMessage(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	// Создаем сообщение
	message := models.Message{
		ConversationID: conversation.ID,
		FromUserID:     user1ID,
		ToUserID:       user2ID,
		Text:           "Hello",
		Status:         models.MessageStatusSent,
	}
	db.Create(&message)

	app := fiber.New()
	messageController := controllers.NewMessageController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Delete("/messages/:id", messageController.DeleteMessage)

	req := httptest.NewRequest("DELETE", "/messages/1", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Проверяем, что сообщение удалено
	var count int64
	db.Model(&models.Message{}).Where("id = ?", message.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestSearchMessages(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	// Создаем сообщения
	messages := []models.Message{
		{
			ConversationID: conversation.ID,
			FromUserID:     user1ID,
			ToUserID:       user2ID,
			Text:           "Hello world",
			Status:         models.MessageStatusSent,
		},
		{
			ConversationID: conversation.ID,
			FromUserID:     user2ID,
			ToUserID:       user1ID,
			Text:           "How are you?",
			Status:         models.MessageStatusSent,
		},
	}

	for _, msg := range messages {
		db.Create(&msg)
	}

	app := fiber.New()
	messageController := controllers.NewMessageController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/messages/search", messageController.SearchMessages)

	req := httptest.NewRequest("GET", "/messages/search?q=hello", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetUnreadMessages(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем диалог
	conversation := models.Conversation{
		UserAID: user1ID,
		UserBID: user2ID,
	}
	db.Create(&conversation)

	// Создаем непрочитанные сообщения
	messages := []models.Message{
		{
			ConversationID: conversation.ID,
			FromUserID:     user2ID,
			ToUserID:       user1ID,
			Text:           "Hello",
			Status:         models.MessageStatusSent,
		},
		{
			ConversationID: conversation.ID,
			FromUserID:     user2ID,
			ToUserID:       user1ID,
			Text:           "How are you?",
			Status:         models.MessageStatusDelivered,
		},
	}

	for _, msg := range messages {
		db.Create(&msg)
	}

	app := fiber.New()
	messageController := controllers.NewMessageController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/messages/unread", messageController.GetUnreadMessages)

	req := httptest.NewRequest("GET", "/messages/unread", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
