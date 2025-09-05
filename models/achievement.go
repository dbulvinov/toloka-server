package models

import (
	"time"

	"gorm.io/gorm"
)

// Achievement представляет модель достижения в системе
type Achievement struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description" gorm:"type:text;not null"`
	IconPath    string    `json:"icon_path" gorm:"not null"`         // Путь к иконке достижения
	Points      int       `json:"points" gorm:"default:0"`           // Очки за достижение
	Category    string    `json:"category" gorm:"default:'general'"` // Категория достижения
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserAchievement представляет связь пользователя с достижением
type UserAchievement struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"not null"`
	AchievementID uint      `json:"achievement_id" gorm:"not null"`
	EarnedAt      time.Time `json:"earned_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Связи
	User        User        `json:"user" gorm:"foreignKey:UserID"`
	Achievement Achievement `json:"achievement" gorm:"foreignKey:AchievementID"`
}

// UserLevel представляет уровень пользователя
type UserLevel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex"`
	Level     int       `json:"level" gorm:"default:1"`
	Points    int       `json:"points" gorm:"default:0"`
	UpdatedAt time.Time `json:"updated_at"`

	// Связи
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// PinnedPost представляет закрепленные посты в профиле
type PinnedPost struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	PostType  string    `json:"post_type" gorm:"not null"` // 'event', 'news', 'comment'
	PostID    uint      `json:"post_id" gorm:"not null"`
	Position  int       `json:"position" gorm:"default:0"` // Позиция в списке закрепленных
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Связи
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// BeforeCreate хук для установки времени создания
func (ua *UserAchievement) BeforeCreate(tx *gorm.DB) error {
	ua.CreatedAt = time.Now()
	ua.UpdatedAt = time.Now()
	ua.EarnedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (ua *UserAchievement) BeforeUpdate(tx *gorm.DB) error {
	ua.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для установки времени создания
func (ul *UserLevel) BeforeCreate(tx *gorm.DB) error {
	ul.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (ul *UserLevel) BeforeUpdate(tx *gorm.DB) error {
	ul.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для установки времени создания
func (pp *PinnedPost) BeforeCreate(tx *gorm.DB) error {
	pp.CreatedAt = time.Now()
	pp.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (pp *PinnedPost) BeforeUpdate(tx *gorm.DB) error {
	pp.UpdatedAt = time.Now()
	return nil
}
