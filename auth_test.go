package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"toloko-backend/controllers"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func setupTestApp() *fiber.App {
	db := setupTestDB()

	app := fiber.New()
	authController := controllers.NewAuthController(db)

	// Настраиваем маршруты
	auth := app.Group("/auth")
	auth.Post("/register", authController.Register)
	auth.Post("/login", authController.Login)
	auth.Post("/recover", authController.Recover)
	auth.Post("/oauth/google", authController.OAuth)
	auth.Post("/oauth/facebook", authController.OAuth)

	return app
}

func TestRegister(t *testing.T) {
	app := setupTestApp()

	tests := []struct {
		name            string
		request         controllers.RegisterRequest
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "Успешная регистрация",
			request: controllers.RegisterRequest{
				Name:            "Тест Пользователь",
				Email:           "test@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
				AcceptTerms:     true,
			},
			expectedStatus:  201,
			expectedSuccess: true,
		},
		{
			name: "Неверный email",
			request: controllers.RegisterRequest{
				Name:            "Тест Пользователь",
				Email:           "invalid-email",
				Password:        "password123",
				ConfirmPassword: "password123",
				AcceptTerms:     true,
			},
			expectedStatus:  400,
			expectedSuccess: false,
		},
		{
			name: "Пароли не совпадают",
			request: controllers.RegisterRequest{
				Name:            "Тест Пользователь",
				Email:           "test2@example.com",
				Password:        "password123",
				ConfirmPassword: "different123",
				AcceptTerms:     true,
			},
			expectedStatus:  400,
			expectedSuccess: false,
		},
		{
			name: "Не приняты условия",
			request: controllers.RegisterRequest{
				Name:            "Тест Пользователь",
				Email:           "test3@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
				AcceptTerms:     false,
			},
			expectedStatus:  400,
			expectedSuccess: false,
		},
		{
			name: "Слишком короткий пароль",
			request: controllers.RegisterRequest{
				Name:            "Тест Пользователь",
				Email:           "test4@example.com",
				Password:        "123",
				ConfirmPassword: "123",
				AcceptTerms:     true,
			},
			expectedStatus:  400,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response controllers.AuthResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)

			if tt.expectedSuccess {
				assert.NotEmpty(t, response.Token)
				assert.NotEmpty(t, response.User.Email)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	app := setupTestApp()

	// Сначала регистрируем пользователя
	registerReq := controllers.RegisterRequest{
		Name:            "Тест Пользователь",
		Email:           "test@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		AcceptTerms:     true,
	}

	jsonData, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	app.Test(req)

	tests := []struct {
		name            string
		request         controllers.LoginRequest
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "Успешный вход",
			request: controllers.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name: "Неверный пароль",
			request: controllers.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			expectedStatus:  401,
			expectedSuccess: false,
		},
		{
			name: "Несуществующий пользователь",
			request: controllers.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			expectedStatus:  401,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response controllers.AuthResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)

			if tt.expectedSuccess {
				assert.NotEmpty(t, response.Token)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	app := setupTestApp()

	tests := []struct {
		name            string
		request         controllers.RecoverRequest
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "Восстановление для существующего пользователя",
			request: controllers.RecoverRequest{
				Email: "test@example.com",
			},
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name: "Восстановление для несуществующего пользователя",
			request: controllers.RecoverRequest{
				Email: "nonexistent@example.com",
			},
			expectedStatus:  200,
			expectedSuccess: true, // Должен возвращать успех для безопасности
		},
		{
			name: "Неверный email",
			request: controllers.RecoverRequest{
				Email: "invalid-email",
			},
			expectedStatus:  400,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/recover", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response controllers.AuthResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSuccess, response.Success)
		})
	}
}

func TestOAuth(t *testing.T) {
	app := setupTestApp()

	tests := []struct {
		name            string
		request         controllers.OAuthRequest
		expectedStatus  int
		expectedSuccess bool
	}{
		{
			name: "Успешная OAuth авторизация Google",
			request: controllers.OAuthRequest{
				Token:    "fake-google-token",
				Provider: "google",
				Name:     "Google User",
				Email:    "google@example.com",
				OAuthID:  "google123",
			},
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name: "Успешная OAuth авторизация Facebook",
			request: controllers.OAuthRequest{
				Token:    "fake-facebook-token",
				Provider: "facebook",
				Name:     "Facebook User",
				Email:    "facebook@example.com",
				OAuthID:  "facebook123",
			},
			expectedStatus:  200,
			expectedSuccess: true,
		},
		{
			name: "Неверный провайдер",
			request: controllers.OAuthRequest{
				Token:    "fake-token",
				Provider: "twitter",
				Name:     "Twitter User",
				Email:    "twitter@example.com",
				OAuthID:  "twitter123",
			},
			expectedStatus:  400,
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/auth/oauth/"+tt.request.Provider, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if resp.StatusCode < 400 {
				var response controllers.AuthResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSuccess, response.Success)

				if tt.expectedSuccess {
					assert.NotEmpty(t, response.Token)
				}
			}
		})
	}
}

func TestJWT(t *testing.T) {
	// Тестируем генерацию и валидацию JWT токенов
	userID := uint(1)
	email := "test@example.com"

	// Генерируем токен
	token, err := utils.GenerateJWT(userID, email)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Валидируем токен
	claims, err := utils.ValidateJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestPasswordHash(t *testing.T) {
	password := "testpassword123"

	// Хэшируем пароль
	hash, err := utils.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Проверяем пароль
	isValid := utils.CheckPasswordHash(password, hash)
	assert.True(t, isValid)

	// Проверяем неверный пароль
	isValid = utils.CheckPasswordHash("wrongpassword", hash)
	assert.False(t, isValid)
}

func TestMain(m *testing.M) {
	// Устанавливаем переменную окружения для JWT
	os.Setenv("JWT_SECRET", "test-secret-key")

	// Запускаем тесты
	code := m.Run()

	// Очищаем
	os.Unsetenv("JWT_SECRET")

	os.Exit(code)
}
