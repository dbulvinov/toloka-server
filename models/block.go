package models

import (
	"time"

	"gorm.io/gorm"
)

// Block представляет блокировку пользователя
type Block struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BlockerID uint      `json:"blocker_id" gorm:"not null;index"`
	BlockedID uint      `json:"blocked_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`

	// Связи
	Blocker User `json:"blocker" gorm:"foreignKey:BlockerID"`
	Blocked User `json:"blocked" gorm:"foreignKey:BlockedID"`
}

// BeforeCreate хук для установки времени создания
func (b *Block) BeforeCreate(tx *gorm.DB) error {
	b.CreatedAt = time.Now()
	return nil
}

// IsBlocked проверяет, заблокирован ли пользователь A пользователем B
func IsBlocked(db *gorm.DB, userAID, userBID uint) (bool, error) {
	var count int64
	err := db.Model(&Block{}).Where("blocker_id = ? AND blocked_id = ?", userAID, userBID).Count(&count).Error
	return count > 0, err
}

// IsBlockedByAny проверяет, заблокирован ли пользователь A любым из пользователей B
func IsBlockedByAny(db *gorm.DB, userAID uint, userBIDs []uint) (bool, error) {
	if len(userBIDs) == 0 {
		return false, nil
	}

	var count int64
	err := db.Model(&Block{}).Where("blocker_id IN ? AND blocked_id = ?", userBIDs, userAID).Count(&count).Error
	return count > 0, err
}
