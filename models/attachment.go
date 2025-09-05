package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Attachment представляет вложение к сообщению
type Attachment struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	MessageID  uint      `json:"message_id" gorm:"not null;index"`
	FilePath   string    `json:"file_path" gorm:"not null"`
	MimeType   string    `json:"mime_type" gorm:"not null"`
	Size       int64     `json:"size" gorm:"not null"`
	UploadedBy uint      `json:"uploaded_by" gorm:"not null;index"`
	CreatedAt  time.Time `json:"created_at"`

	// Связи
	Message  Message `json:"message" gorm:"foreignKey:MessageID"`
	Uploader User    `json:"uploader" gorm:"foreignKey:UploadedBy"`
}

// BeforeCreate хук для установки времени создания
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	a.CreatedAt = time.Now()
	return nil
}

// IsImage проверяет, является ли вложение изображением
func (a *Attachment) IsImage() bool {
	return a.MimeType == "image/jpeg" || a.MimeType == "image/png" || a.MimeType == "image/gif"
}

// GetFileName возвращает имя файла из пути
func (a *Attachment) GetFileName() string {
	// Простая реализация - можно улучшить
	parts := strings.Split(a.FilePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
