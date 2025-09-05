package models

import (
	"time"

	"gorm.io/gorm"
)

// Inventory представляет модель инвентаря в системе
type Inventory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null;size:255;uniqueIndex"`
	IsCustom  bool      `json:"is_custom" gorm:"default:false"` // true - пользовательский, false - системный
	CreatedBy uint      `json:"created_by" gorm:"default:0"`    // ID пользователя, создавшего инвентарь (0 для системного)
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Связи
	Creator *User `json:"creator" gorm:"foreignKey:CreatedBy"`
}

// BeforeCreate хук для установки времени создания
func (i *Inventory) BeforeCreate(tx *gorm.DB) error {
	i.CreatedAt = time.Now()
	i.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (i *Inventory) BeforeUpdate(tx *gorm.DB) error {
	i.UpdatedAt = time.Now()
	return nil
}
