package main

import (
	"bytes"
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

func setupUserTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Achievement{}, &models.UserAchievement{}, &models.UserLevel{}, &models.PinnedPost{})
	return db
}

func createUserTestUser(db *gorm.DB, name, email string) *models.User {
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: "hashedpassword",
		IsActive:     true,
		Avatar:       "https://example.com/avatar.jpg",
		Bio:          "Test bio",
		Location:     "Test City",
		Website:      "https://test.com",
		IsPublic:     true,
	}
	db.Create(user)
	return user
}

func createTestAchievement(db *gorm.DB) *models.Achievement {
	achievement := &models.Achievement{
		Name:        "Test Achievement",
		Description: "Test achievement description",
		IconPath:    "/icons/test.png",
		Points:      50,
		Category:    "test",
		IsActive:    true,
	}
	db.Create(achievement)
	return achievement
}

func TestGetProfile(t *testing.T) {
	db := setupUserTestDB()
	userController := controllers.NewUserController(db)
	app := fiber.New()
	app.Get("/users/:id", userController.GetProfile)

	// Создаем тестового пользователя
	_ = createUserTestUser(db, "Test User", "test@example.com")

	// Тест успешного получения профиля
	req := httptest.NewRequest("GET", "/users/1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["user"])
}

func TestUpdateProfile(t *testing.T) {
	db := setupUserTestDB()
	userController := controllers.NewUserController(db)
	app := fiber.New()
	app.Put("/users/:id", userController.UpdateProfile)

	// Создаем тестового пользователя
	user := createUserTestUser(db, "Test User", "test@example.com")

	// Создаем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	assert.NoError(t, err)

	// Тест обновления профиля
	updateData := map[string]interface{}{
		"name":      "Updated Name",
		"bio":       "Updated bio",
		"location":  "Updated City",
		"website":   "https://updated.com",
		"is_public": true,
	}

	jsonData, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PUT", "/users/1", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["user"])

	// Проверяем, что данные обновились
	userData := result["user"].(map[string]interface{})
	assert.Equal(t, "Updated Name", userData["name"])
	assert.Equal(t, "Updated bio", userData["bio"])
}

func TestGetUserEvents(t *testing.T) {
	db := setupUserTestDB()
	userController := controllers.NewUserController(db)
	app := fiber.New()
	app.Get("/users/:id/events", userController.GetUserEvents)

	// Создаем тестового пользователя
	createUserTestUser(db, "Test User", "test@example.com")

	// Тест получения событий пользователя
	req := httptest.NewRequest("GET", "/users/1/events", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["events"])
}

func TestGetUserCommunities(t *testing.T) {
	db := setupUserTestDB()
	userController := controllers.NewUserController(db)
	app := fiber.New()
	app.Get("/users/:id/communities", userController.GetUserCommunities)

	// Создаем тестового пользователя
	createUserTestUser(db, "Test User", "test@example.com")

	// Тест получения сообществ пользователя
	req := httptest.NewRequest("GET", "/users/1/communities", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["communities"])
}

func TestGetUserAchievements(t *testing.T) {
	db := setupUserTestDB()
	achievementController := controllers.NewAchievementController(db)
	app := fiber.New()
	app.Get("/achievements/user/:id", achievementController.GetUserAchievements)

	// Создаем тестового пользователя
	user := createUserTestUser(db, "Test User", "test@example.com")

	// Создаем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	assert.NoError(t, err)

	// Тест получения достижений пользователя
	req := httptest.NewRequest("GET", "/achievements/user/1", nil)
	req.Header.Set("Authorization", token)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["achievements"])
}

func TestAwardAchievement(t *testing.T) {
	db := setupUserTestDB()
	achievementController := controllers.NewAchievementController(db)
	app := fiber.New()
	app.Post("/achievements/user/:user_id/award/:achievement_id", achievementController.AwardAchievement)

	// Создаем тестового пользователя и достижение
	user := createUserTestUser(db, "Test User", "test@example.com")
	_ = createTestAchievement(db)

	// Создаем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	assert.NoError(t, err)

	// Тест награждения достижением
	req := httptest.NewRequest("POST", "/achievements/user/1/award/1", nil)
	req.Header.Set("Authorization", token)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["user_achievement"])
}

func TestGetUserLevel(t *testing.T) {
	db := setupUserTestDB()
	levelController := controllers.NewLevelController(db)
	app := fiber.New()
	app.Get("/levels/user/:id", levelController.GetUserLevel)

	// Создаем тестового пользователя
	user := createUserTestUser(db, "Test User", "test@example.com")

	// Создаем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	assert.NoError(t, err)

	// Тест получения уровня пользователя
	req := httptest.NewRequest("GET", "/levels/user/1", nil)
	req.Header.Set("Authorization", token)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["user_level"])
}

func TestAddPoints(t *testing.T) {
	db := setupUserTestDB()
	levelController := controllers.NewLevelController(db)
	app := fiber.New()
	app.Post("/levels/user/:id/add-points", levelController.AddPoints)

	// Создаем тестового пользователя
	user := createUserTestUser(db, "Test User", "test@example.com")

	// Создаем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	assert.NoError(t, err)

	// Тест добавления очков
	addPointsData := map[string]interface{}{
		"points": 50,
		"reason": "Test points",
	}

	jsonData, _ := json.Marshal(addPointsData)
	req := httptest.NewRequest("POST", "/levels/user/1/add-points", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["user_level"])
}

func TestGetLeaderboard(t *testing.T) {
	db := setupUserTestDB()
	levelController := controllers.NewLevelController(db)
	app := fiber.New()
	app.Get("/levels/leaderboard", levelController.GetLeaderboard)

	// Создаем тестовых пользователей с уровнями
	user1 := createUserTestUser(db, "Test User 1", "test1@example.com")
	user2 := createUserTestUser(db, "Test User 2", "test2@example.com")

	// Создаем уровни пользователей
	level1 := &models.UserLevel{
		UserID: user1.ID,
		Level:  2,
		Points: 150,
	}
	level2 := &models.UserLevel{
		UserID: user2.ID,
		Level:  1,
		Points: 50,
	}
	db.Create(level1)
	db.Create(level2)

	// Тест получения таблицы лидеров
	req := httptest.NewRequest("GET", "/levels/leaderboard", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.False(t, result["error"].(bool))
	assert.NotNil(t, result["leaderboard"])
}
