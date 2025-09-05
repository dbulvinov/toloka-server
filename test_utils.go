package main

import (
	"time"
	"toloko-backend/models"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB создает тестовую базу данных в памяти
func setupTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Conversation{}, &models.Message{}, &models.Attachment{}, &models.Block{}, &models.UserPresence{})
	return db
}

// createTestUsers создает тестовых пользователей и возвращает их ID
func createTestUsers(db *gorm.DB) (uint, uint) {
	user1 := models.User{
		Name:         "Test User 1",
		Email:        "user1@test.com",
		PasswordHash: "hash1",
	}
	user2 := models.User{
		Name:         "Test User 2",
		Email:        "user2@test.com",
		PasswordHash: "hash2",
	}

	db.Create(&user1)
	db.Create(&user2)

	return user1.ID, user2.ID
}

// generateTestJWT создает тестовый JWT токен для указанного пользователя
func generateTestJWT(userID uint) string {
	secretKey := "toloko-secret-key-change-in-production"
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))
	return tokenString
}

// validateTestJWT проверяет тестовый JWT токен и возвращает user_id
func validateTestJWT(tokenString string) (uint, error) {
	secretKey := "toloko-secret-key-change-in-production"

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, ok := claims["user_id"].(float64); ok {
			return uint(userID), nil
		}
	}

	return 0, jwt.ErrTokenMalformed
}
