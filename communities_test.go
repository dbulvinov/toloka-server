package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"toloko-backend/controllers"
	"toloko-backend/models"
	"toloko-backend/routes"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupCommunitiesTestDB создает тестовую базу данных в памяти для тестов сообществ
func setupCommunitiesTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}

	// Автомиграция
	db.AutoMigrate(&models.User{}, &models.Community{}, &models.CommunityRole{}, &models.News{}, &models.Comment{}, &models.NewsLike{})

	return db
}

// createCommunitiesTestUser создает тестового пользователя для тестов сообществ
func createCommunitiesTestUser(db *gorm.DB) (*models.User, string) {
	user := models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		IsActive:     true,
	}

	db.Create(&user)

	token, _ := utils.GenerateJWT(user.ID, user.Email)
	return &user, token
}

// createCommunitiesTestCommunity создает тестовое сообщество
func createCommunitiesTestCommunity(db *gorm.DB, creatorID uint) *models.Community {
	community := models.Community{
		CreatorID:   creatorID,
		Name:        "Test Community",
		Description: "Test Description",
		City:        "Test City",
	}

	db.Create(&community)

	// Создаем роль администратора
	role := models.CommunityRole{
		CommunityID: community.ID,
		UserID:      creatorID,
		Role:        "admin",
	}
	db.Create(&role)

	return &community
}

// setupCommunitiesTestApp создает тестовое приложение для тестов сообществ
func setupCommunitiesTestApp(db *gorm.DB) *fiber.App {
	app := fiber.New()

	// Инициализация контроллеров
	communityController := controllers.NewCommunityController(db)
	newsController := controllers.NewNewsController(db)
	commentController := controllers.NewCommentController(db)

	// Настройка маршрутов
	routes.SetupCommunityRoutes(app, communityController)
	routes.SetupNewsRoutes(app, newsController)
	routes.SetupCommentRoutes(app, commentController)

	return app
}

func TestCreateCommunity(t *testing.T) {
	db := setupCommunitiesTestDB()
	app := setupCommunitiesTestApp(db)
	user, token := createCommunitiesTestUser(db)

	// Тестовые данные
	communityData := controllers.CreateCommunityRequest{
		Name:        "New Community",
		Description: "New Description",
		City:        "New City",
	}

	jsonData, _ := json.Marshal(communityData)

	// Создаем запрос
	req := httptest.NewRequest("POST", "/communities", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Проверяем, что сообщество создано в базе
	var community models.Community
	err = db.Where("name = ?", "New Community").First(&community).Error
	assert.NoError(t, err)
	assert.Equal(t, user.ID, community.CreatorID)
	assert.Equal(t, "New Community", community.Name)

	// Проверяем, что создана роль администратора
	var role models.CommunityRole
	err = db.Where("community_id = ? AND user_id = ?", community.ID, user.ID).First(&role).Error
	assert.NoError(t, err)
	assert.Equal(t, "admin", role.Role)
}

func TestGetCommunities(t *testing.T) {
	db := setupCommunitiesTestDB()
	app := setupCommunitiesTestApp(db)
	user, _ := createCommunitiesTestUser(db)

	// Создаем тестовые сообщества
	createCommunitiesTestCommunity(db, user.ID)
	createCommunitiesTestCommunity(db, user.ID)

	// Создаем запрос
	req := httptest.NewRequest("GET", "/communities", nil)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Проверяем ответ
	var response controllers.CommunitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, int64(2), response.Total)
	assert.Len(t, response.Data, 2)
}

func TestCreateNews(t *testing.T) {
	db := setupCommunitiesTestDB()
	app := setupCommunitiesTestApp(db)
	user, token := createCommunitiesTestUser(db)

	// Создаем тестовое сообщество
	community := createCommunitiesTestCommunity(db, user.ID)

	// Тестовые данные
	newsData := controllers.CreateNewsRequest{
		Content:   "Test news content",
		PhotoPath: "/path/to/photo.jpg",
	}

	jsonData, _ := json.Marshal(newsData)

	// Создаем запрос
	req := httptest.NewRequest("POST", "/communities/1/news", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Проверяем, что новость создана в базе
	var news models.News
	err = db.Where("community_id = ? AND author_id = ?", community.ID, user.ID).First(&news).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test news content", news.Content)
	assert.Equal(t, "/path/to/photo.jpg", news.PhotoPath)
}

func TestCreateComment(t *testing.T) {
	db := setupCommunitiesTestDB()
	app := setupCommunitiesTestApp(db)
	user, token := createCommunitiesTestUser(db)

	// Создаем тестовое сообщество и новость
	community := createCommunitiesTestCommunity(db, user.ID)
	news := models.News{
		CommunityID: community.ID,
		AuthorID:    user.ID,
		Content:     "Test news",
	}
	db.Create(&news)

	// Тестовые данные
	commentData := controllers.CreateCommentRequest{
		Content: "Test comment",
	}

	jsonData, _ := json.Marshal(commentData)

	// Создаем запрос
	req := httptest.NewRequest("POST", "/news/1/comments", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Выполняем запрос
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Проверяем, что комментарий создан в базе
	var comment models.Comment
	err = db.Where("news_id = ? AND author_id = ?", news.ID, user.ID).First(&comment).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test comment", comment.Content)
}
