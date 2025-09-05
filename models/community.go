package models

import (
	"time"

	"gorm.io/gorm"
)

// Community представляет модель сообщества
type Community struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CreatorID   uint      `json:"creator_id" gorm:"not null"`
	Name        string    `json:"name" gorm:"not null;size:100"`
	Description string    `json:"description" gorm:"size:500"`
	City        string    `json:"city" gorm:"not null;size:100"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Связи
	Creator User            `json:"creator" gorm:"foreignKey:CreatorID"`
	Roles   []CommunityRole `json:"roles" gorm:"foreignKey:CommunityID"`
	News    []News          `json:"news" gorm:"foreignKey:CommunityID"`
}

// CommunityRole представляет роли пользователей в сообществе
type CommunityRole struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CommunityID uint      `json:"community_id" gorm:"not null"`
	UserID      uint      `json:"user_id" gorm:"not null"`
	Role        string    `json:"role" gorm:"not null;size:20"` // "admin", "moderator", "member"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Связи
	Community Community `json:"community" gorm:"foreignKey:CommunityID"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
}

// News представляет новость в сообществе
type News struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CommunityID uint      `json:"community_id" gorm:"not null"`
	AuthorID    uint      `json:"author_id" gorm:"not null"`
	Content     string    `json:"content" gorm:"not null;type:text"`
	PhotoPath   string    `json:"photo_path" gorm:"size:255"`
	LikesCount  int       `json:"likes_count" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Связи
	Community Community  `json:"community" gorm:"foreignKey:CommunityID"`
	Author    User       `json:"author" gorm:"foreignKey:AuthorID"`
	Comments  []Comment  `json:"comments" gorm:"foreignKey:NewsID"`
	Likes     []NewsLike `json:"likes" gorm:"foreignKey:NewsID"`
}

// Comment представляет комментарий к новости
type Comment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	NewsID    uint      `json:"news_id" gorm:"not null"`
	AuthorID  uint      `json:"author_id" gorm:"not null"`
	ParentID  *uint     `json:"parent_id"` // Для дерева комментариев
	Content   string    `json:"content" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Связи
	News     News      `json:"news" gorm:"foreignKey:NewsID"`
	Author   User      `json:"author" gorm:"foreignKey:AuthorID"`
	Parent   *Comment  `json:"parent" gorm:"foreignKey:ParentID"`
	Children []Comment `json:"children" gorm:"foreignKey:ParentID"`
}

// NewsLike представляет лайк новости
type NewsLike struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	NewsID    uint      `json:"news_id" gorm:"not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`

	// Связи
	News News `json:"news" gorm:"foreignKey:NewsID"`
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// BeforeCreate хук для установки времени создания
func (c *Community) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (c *Community) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для установки времени создания
func (cr *CommunityRole) BeforeCreate(tx *gorm.DB) error {
	cr.CreatedAt = time.Now()
	cr.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (cr *CommunityRole) BeforeUpdate(tx *gorm.DB) error {
	cr.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для установки времени создания
func (n *News) BeforeCreate(tx *gorm.DB) error {
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (n *News) BeforeUpdate(tx *gorm.DB) error {
	n.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для установки времени создания
func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (c *Comment) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для установки времени создания
func (nl *NewsLike) BeforeCreate(tx *gorm.DB) error {
	nl.CreatedAt = time.Now()
	return nil
}

// IsAdmin проверяет, является ли пользователь администратором сообщества
func (cr *CommunityRole) IsAdmin() bool {
	return cr.Role == "admin"
}

// IsModerator проверяет, является ли пользователь модератором сообщества
func (cr *CommunityRole) IsModerator() bool {
	return cr.Role == "moderator"
}

// IsMember проверяет, является ли пользователь участником сообщества
func (cr *CommunityRole) IsMember() bool {
	return cr.Role == "member"
}

// CanManageCommunity проверяет, может ли пользователь управлять сообществом
func (cr *CommunityRole) CanManageCommunity() bool {
	return cr.IsAdmin() || cr.IsModerator()
}

// CanManageNews проверяет, может ли пользователь управлять новостями
func (cr *CommunityRole) CanManageNews() bool {
	return cr.IsAdmin() || cr.IsModerator()
}
