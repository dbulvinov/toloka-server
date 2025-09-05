package utils

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Claims представляет структуру JWT токена
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateJWT создает JWT токен для пользователя
func GenerateJWT(userID uint, email string) (string, error) {
	// Получаем секретный ключ из переменной окружения или используем дефолтный
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "toloko-secret-key-change-in-production"
	}

	// Создаем claims
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Токен действителен 24 часа
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT проверяет и парсит JWT токен
func ValidateJWT(tokenString string) (*Claims, error) {
	// Получаем секретный ключ
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "toloko-secret-key-change-in-production"
	}

	// Парсим токен
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secretKey), nil
	}, jwt.WithLeeway(5*time.Minute))

	if err != nil {
		return nil, err
	}

	// Проверяем валидность токена
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenMalformed
}

// AuthMiddleware middleware для проверки JWT токена
func AuthMiddleware(c *fiber.Ctx) error {
	// Получаем токен из заголовка Authorization
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	// Проверяем формат Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid authorization header format",
		})
	}

	tokenString := tokenParts[1]

	// Валидируем токен
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Сохраняем информацию о пользователе в контексте
	c.Locals("user_id", claims.UserID)
	c.Locals("user_email", claims.Email)

	return c.Next()
}
