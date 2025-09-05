package models

import (
	"time"

	"gorm.io/gorm"
)

// Conversation представляет диалог между двумя пользователями
type Conversation struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserAID       uint      `json:"user_a_id" gorm:"not null;index"`
	UserBID       uint      `json:"user_b_id" gorm:"not null;index"`
	LastMessageID *uint     `json:"last_message_id" gorm:"index"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`

	// Связи
	UserA       User      `json:"user_a" gorm:"foreignKey:UserAID"`
	UserB       User      `json:"user_b" gorm:"foreignKey:UserBID"`
	LastMessage *Message  `json:"last_message" gorm:"foreignKey:LastMessageID"`
	Messages    []Message `json:"messages" gorm:"foreignKey:ConversationID"`
}

// BeforeCreate хук для установки времени создания
func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (c *Conversation) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// GetOtherUserID возвращает ID собеседника для данного пользователя
func (c *Conversation) GetOtherUserID(userID uint) uint {
	if c.UserAID == userID {
		return c.UserBID
	}
	return c.UserAID
}

// GetOtherUser возвращает собеседника для данного пользователя
func (c *Conversation) GetOtherUser(userID uint) User {
	if c.UserAID == userID {
		return c.UserB
	}
	return c.UserA
}
