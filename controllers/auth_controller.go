package controllers

import (
	"regexp"
	"strings"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AuthController контроллер для аутентификации
type AuthController struct {
	DB *gorm.DB
}

// NewAuthController создает новый экземпляр AuthController
func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
}

// RegisterRequest структура запроса регистрации
type RegisterRequest struct {
	Name            string `json:"name" validate:"required,min=2,max=50"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	AcceptTerms     bool   `json:"accept_terms" validate:"required"`
}

// LoginRequest структура запроса входа
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RecoverRequest структура запроса восстановления пароля
type RecoverRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// OAuthRequest структура запроса OAuth
type OAuthRequest struct {
	Token    string `json:"token" validate:"required"`
	Provider string `json:"provider" validate:"required,oneof=google facebook"`
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	OAuthID  string `json:"oauth_id" validate:"required"`
}

// AuthResponse структура ответа аутентификации
type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	User    struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user,omitempty"`
}

// Register обрабатывает регистрацию пользователя
func (ac *AuthController) Register(c *fiber.Ctx) error {
	var req RegisterRequest

	// Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := ac.validateRegisterRequest(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Проверяем, существует ли пользователь
	var existingUser models.User
	if err := ac.DB.Where("email = ?", strings.ToLower(req.Email)).First(&existingUser).Error; err == nil {
		return c.Status(409).JSON(AuthResponse{
			Success: false,
			Message: "Пользователь с таким email уже существует",
		})
	}

	// Хэшируем пароль
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Message: "Ошибка при создании пользователя",
		})
	}

	// Создаем пользователя
	user := models.User{
		Name:         req.Name,
		Email:        strings.ToLower(req.Email),
		PasswordHash: hashedPassword,
		IsActive:     true,
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Message: "Ошибка при создании пользователя",
		})
	}

	// Генерируем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Message: "Ошибка при создании токена",
		})
	}

	return c.Status(201).JSON(AuthResponse{
		Success: true,
		Message: "Пользователь успешно зарегистрирован",
		Token:   token,
		User: struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// Login обрабатывает вход пользователя
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req LoginRequest

	// Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := ac.validateLoginRequest(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Ищем пользователя
	var user models.User
	if err := ac.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		return c.Status(401).JSON(AuthResponse{
			Success: false,
			Message: "Неверный email или пароль",
		})
	}

	// Проверяем пароль
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return c.Status(401).JSON(AuthResponse{
			Success: false,
			Message: "Неверный email или пароль",
		})
	}

	// Проверяем активность пользователя
	if !user.IsActive {
		return c.Status(401).JSON(AuthResponse{
			Success: false,
			Message: "Аккаунт заблокирован",
		})
	}

	// Генерируем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Message: "Ошибка при создании токена",
		})
	}

	return c.JSON(AuthResponse{
		Success: true,
		Message: "Успешный вход в систему",
		Token:   token,
		User: struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// Recover обрабатывает запрос на восстановление пароля
func (ac *AuthController) Recover(c *fiber.Ctx) error {
	var req RecoverRequest

	// Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация email
	if !ac.isValidEmail(req.Email) {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: "Неверный формат email",
		})
	}

	// Проверяем, существует ли пользователь
	var user models.User
	if err := ac.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		// Возвращаем успех даже если пользователь не найден (безопасность)
		return c.JSON(AuthResponse{
			Success: true,
			Message: "Если пользователь с таким email существует, на него отправлено письмо с инструкциями",
		})
	}

	// В реальном приложении здесь бы отправлялось письмо
	// Для демонстрации просто возвращаем успех
	return c.JSON(AuthResponse{
		Success: true,
		Message: "Если пользователь с таким email существует, на него отправлено письмо с инструкциями",
	})
}

// OAuth обрабатывает OAuth авторизацию
func (ac *AuthController) OAuth(c *fiber.Ctx) error {
	var req OAuthRequest

	// Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: "Неверный формат данных",
		})
	}

	// Валидация
	if err := ac.validateOAuthRequest(&req); err != nil {
		return c.Status(400).JSON(AuthResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	// Ищем пользователя по OAuth ID
	var user models.User
	err := ac.DB.Where("oauth_provider = ? AND oauth_id = ?", req.Provider, req.OAuthID).First(&user).Error

	if err != nil {
		// Пользователь не найден, создаем нового
		user = models.User{
			Name:          req.Name,
			Email:         strings.ToLower(req.Email),
			OAuthProvider: req.Provider,
			OAuthID:       req.OAuthID,
			IsActive:      true,
		}

		if err := ac.DB.Create(&user).Error; err != nil {
			return c.Status(500).JSON(AuthResponse{
				Success: false,
				Message: "Ошибка при создании пользователя",
			})
		}
	}

	// Генерируем JWT токен
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(500).JSON(AuthResponse{
			Success: false,
			Message: "Ошибка при создании токена",
		})
	}

	return c.JSON(AuthResponse{
		Success: true,
		Message: "Успешная авторизация через " + req.Provider,
		Token:   token,
		User: struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// Вспомогательные методы валидации

func (ac *AuthController) validateRegisterRequest(req *RegisterRequest) error {
	if req.Name == "" {
		return fiber.NewError(400, "Имя обязательно")
	}
	if len(req.Name) < 2 || len(req.Name) > 50 {
		return fiber.NewError(400, "Имя должно содержать от 2 до 50 символов")
	}
	if !ac.isValidEmail(req.Email) {
		return fiber.NewError(400, "Неверный формат email")
	}
	if len(req.Password) < 6 {
		return fiber.NewError(400, "Пароль должен содержать минимум 6 символов")
	}
	if req.Password != req.ConfirmPassword {
		return fiber.NewError(400, "Пароли не совпадают")
	}
	if !req.AcceptTerms {
		return fiber.NewError(400, "Необходимо принять условия использования")
	}
	return nil
}

func (ac *AuthController) validateLoginRequest(req *LoginRequest) error {
	if !ac.isValidEmail(req.Email) {
		return fiber.NewError(400, "Неверный формат email")
	}
	if req.Password == "" {
		return fiber.NewError(400, "Пароль обязателен")
	}
	return nil
}

func (ac *AuthController) validateOAuthRequest(req *OAuthRequest) error {
	if req.Token == "" {
		return fiber.NewError(400, "OAuth токен обязателен")
	}
	if req.Provider != "google" && req.Provider != "facebook" {
		return fiber.NewError(400, "Неподдерживаемый OAuth провайдер")
	}
	if req.Name == "" {
		return fiber.NewError(400, "Имя обязательно")
	}
	if !ac.isValidEmail(req.Email) {
		return fiber.NewError(400, "Неверный формат email")
	}
	if req.OAuthID == "" {
		return fiber.NewError(400, "OAuth ID обязателен")
	}
	return nil
}

func (ac *AuthController) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
