package main

import (
	"testing"
	"time"

	"toloko-backend/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	// Тестируем генерацию токена
	token, err := utils.GenerateJWT(1, "test@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Тестируем валидацию токена
	claims, err := utils.ValidateJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)

	// Тестируем наш тестовый токен
	testToken := generateTestJWT(1)
	assert.NotEmpty(t, testToken)

	// Тестируем валидацию тестового токена
	testUserID, err := validateTestJWT(testToken)
	if err != nil {
		t.Logf("JWT validation error: %v", err)
		t.Logf("Test token: %s", testToken)
	} else {
		t.Logf("JWT validation successful: userID=%d", testUserID)
		assert.Equal(t, uint(1), testUserID)
	}
}

func TestJWTWithDifferentVersions(t *testing.T) {
	// Создаем токен с JWT v5
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
		"email":   "test@example.com",
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte("toloko-secret-key-change-in-production"))
	assert.NoError(t, err)

	// Пытаемся валидировать с utils.ValidateJWT
	claims, err := utils.ValidateJWT(tokenString)
	if err != nil {
		t.Logf("JWT validation error: %v", err)
		t.Logf("Token: %s", tokenString)
	} else {
		t.Logf("JWT validation successful: %+v", claims)
	}
}
