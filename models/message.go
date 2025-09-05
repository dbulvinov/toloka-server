package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// MessageStatus представляет статус сообщения
type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

// Message представляет сообщение в диалоге
type Message struct {
	ID              uint          `json:"id" gorm:"primaryKey"`
	ConversationID  uint          `json:"conversation_id" gorm:"not null;index"`
	FromUserID      uint          `json:"from_user_id" gorm:"not null;index"`
	ToUserID        uint          `json:"to_user_id" gorm:"not null;index"`
	Text            string        `json:"text" gorm:"type:text"`
	AttachmentsJSON string        `json:"-" gorm:"column:attachments_json;type:text"`
	Attachments     []Attachment  `json:"attachments" gorm:"foreignKey:MessageID"`
	Status          MessageStatus `json:"status" gorm:"default:'sent'"`
	TempID          string        `json:"temp_id" gorm:"index"` // Временный ID для клиента
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`

	// Связи
	Conversation Conversation `json:"conversation" gorm:"foreignKey:ConversationID"`
	FromUser     User         `json:"from_user" gorm:"foreignKey:FromUserID"`
	ToUser       User         `json:"to_user" gorm:"foreignKey:ToUserID"`
}

// AttachmentData представляет данные о вложениях в JSON формате
type AttachmentData struct {
	ID       uint   `json:"id"`
	FilePath string `json:"file_path"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// BeforeCreate хук для установки времени создания
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (m *Message) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// SetAttachmentsData устанавливает данные о вложениях в JSON формате
func (m *Message) SetAttachmentsData(attachments []AttachmentData) error {
	data, err := json.Marshal(attachments)
	if err != nil {
		return err
	}
	m.AttachmentsJSON = string(data)
	return nil
}

// GetAttachmentsData получает данные о вложениях из JSON формата
func (m *Message) GetAttachmentsData() ([]AttachmentData, error) {
	var attachments []AttachmentData
	if m.AttachmentsJSON == "" {
		return attachments, nil
	}
	err := json.Unmarshal([]byte(m.AttachmentsJSON), &attachments)
	return attachments, err
}

// IsFromUser проверяет, отправлено ли сообщение данным пользователем
func (m *Message) IsFromUser(userID uint) bool {
	return m.FromUserID == userID
}

// IsToUser проверяет, адресовано ли сообщение данному пользователю
func (m *Message) IsToUser(userID uint) bool {
	return m.ToUserID == userID
}
