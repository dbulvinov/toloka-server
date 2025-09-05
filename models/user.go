package models

import (
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User представляет модель пользователя в системе
type User struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	Name          string `json:"name" gorm:"not null"`
	Email         string `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash  string `json:"-" gorm:"not null"` // Скрываем хэш пароля в JSON
	OAuthProvider string `json:"oauth_provider" gorm:"default:''"`
	OAuthID       string `json:"oauth_id" gorm:"default:''"`
	IsActive      bool   `json:"is_active" gorm:"default:true"`
	// Поля профиля
	Avatar    string    `json:"avatar" gorm:"default:''"`        // URL аватара
	Bio       string    `json:"bio" gorm:"type:text;default:''"` // Описание профиля
	Location  string    `json:"location" gorm:"default:''"`      // Местоположение
	Website   string    `json:"website" gorm:"default:''"`       // Веб-сайт
	IsPublic  bool      `json:"is_public" gorm:"default:true"`   // Публичный профиль
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InitDB инициализирует подключение к базе данных
func InitDB() (*gorm.DB, error) {
	// Проверяем переменную окружения для выбора базы данных
	databaseURL := os.Getenv("DATABASE_URL")
	
	if databaseURL != "" {
		// Используем PostgreSQL для продакшена
		db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		return db, nil
	}
	
	// Используем SQLite для разработки
	db, err := gorm.Open(sqlite.Open("toloko.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// BeforeCreate хук для установки времени создания
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}
