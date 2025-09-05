package main

import (
	"net/http/httptest"
	"testing"

	"toloko-backend/controllers"
	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestBlockUser(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Post("/blocks/:user_id", blockController.BlockUser)

	req := httptest.NewRequest("POST", "/blocks/2", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Проверяем, что блокировка создана
	var block models.Block
	err = db.Where("blocker_id = ? AND blocked_id = ?", user1ID, user2ID).First(&block).Error
	assert.NoError(t, err)
	assert.Equal(t, user1ID, block.BlockerID)
	assert.Equal(t, user2ID, block.BlockedID)
}

func TestUnblockUser(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем блокировку
	block := models.Block{
		BlockerID: user1ID,
		BlockedID: user2ID,
	}
	db.Create(&block)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Delete("/blocks/:user_id", blockController.UnblockUser)

	req := httptest.NewRequest("DELETE", "/blocks/2", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Проверяем, что блокировка удалена
	var count int64
	db.Model(&models.Block{}).Where("blocker_id = ? AND blocked_id = ?", user1ID, user2ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestGetBlockedUsers(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем блокировку
	block := models.Block{
		BlockerID: user1ID,
		BlockedID: user2ID,
	}
	db.Create(&block)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/blocks", blockController.GetBlockedUsers)

	req := httptest.NewRequest("GET", "/blocks", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestIsBlocked(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем блокировку
	block := models.Block{
		BlockerID: user1ID,
		BlockedID: user2ID,
	}
	db.Create(&block)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/blocks/:user_id", blockController.IsBlocked)

	req := httptest.NewRequest("GET", "/blocks/2", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetBlockStats(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем блокировку
	block := models.Block{
		BlockerID: user1ID,
		BlockedID: user2ID,
	}
	db.Create(&block)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Get("/blocks/stats", blockController.GetBlockStats)

	req := httptest.NewRequest("GET", "/blocks/stats", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestBlockUserCannotBlockSelf(t *testing.T) {
	db := setupTestDB()
	user1ID, _ := createTestUsers(db)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Post("/blocks/:user_id", blockController.BlockUser)

	req := httptest.NewRequest("POST", "/blocks/1", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestBlockUserAlreadyBlocked(t *testing.T) {
	db := setupTestDB()
	user1ID, user2ID := createTestUsers(db)

	// Создаем блокировку
	block := models.Block{
		BlockerID: user1ID,
		BlockedID: user2ID,
	}
	db.Create(&block)

	app := fiber.New()
	blockController := controllers.NewBlockController(db)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", user1ID)
		return c.Next()
	})

	app.Post("/blocks/:user_id", blockController.BlockUser)

	req := httptest.NewRequest("POST", "/blocks/2", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)
}
