package models

import (
	"time"

	"gorm.io/gorm"
)

// UserPresence представляет статус присутствия пользователя
type UserPresence struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	IsOnline   bool      `json:"is_online" gorm:"default:false"`
	LastSeenAt time.Time `json:"last_seen_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Связи
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// BeforeCreate хук для установки времени создания
func (p *UserPresence) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.LastSeenAt = now
	p.UpdatedAt = now
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (p *UserPresence) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}

// SetOnline устанавливает пользователя как онлайн
func (p *UserPresence) SetOnline() {
	p.IsOnline = true
	p.LastSeenAt = time.Now()
	p.UpdatedAt = time.Now()
}

// SetOffline устанавливает пользователя как офлайн
func (p *UserPresence) SetOffline() {
	p.IsOnline = false
	p.LastSeenAt = time.Now()
	p.UpdatedAt = time.Now()
}

// UpdatePresence обновляет статус присутствия пользователя
func UpdatePresence(db *gorm.DB, userID uint, isOnline bool) error {
	var presence UserPresence

	// Пытаемся найти существующую запись
	err := db.Where("user_id = ?", userID).First(&presence).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Создаем новую запись
			presence = UserPresence{
				UserID: userID,
			}
		} else {
			return err
		}
	}

	// Обновляем статус
	if isOnline {
		presence.SetOnline()
	} else {
		presence.SetOffline()
	}

	// Сохраняем
	return db.Save(&presence).Error
}

// GetPresence получает статус присутствия пользователя
func GetPresence(db *gorm.DB, userID uint) (*UserPresence, error) {
	var presence UserPresence
	err := db.Where("user_id = ?", userID).First(&presence).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Возвращаем офлайн статус по умолчанию
			return &UserPresence{
				UserID:     userID,
				IsOnline:   false,
				LastSeenAt: time.Time{},
			}, nil
		}
		return nil, err
	}
	return &presence, nil
}
