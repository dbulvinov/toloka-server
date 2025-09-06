package models

import (
	"time"

	"gorm.io/gorm"
)

// Event представляет модель ивента в системе
type Event struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	CreatorID        uint      `json:"creator_id" gorm:"not null"`
	Title            string    `json:"title" gorm:"not null;size:255"`
	Description      string    `json:"description" gorm:"type:text"`
	Latitude         float64   `json:"latitude" gorm:"not null"`
	Longitude        float64   `json:"longitude" gorm:"not null"`
	LocationName     string    `json:"location_name" gorm:"size:255"` // Название места
	Address          string    `json:"address" gorm:"type:text"`      // Полный адрес
	City             string    `json:"city" gorm:"size:100"`          // Город
	StartTime        time.Time `json:"start_time" gorm:"not null"`
	EndTime          time.Time `json:"end_time" gorm:"not null"`
	JoinMode         string    `json:"join_mode" gorm:"not null;default:'free'"` // 'free' или 'approval'
	MinParticipants  int       `json:"min_participants" gorm:"default:1"`
	MaxParticipants  int       `json:"max_participants" gorm:"default:50"`
	EventType        string    `json:"event_type" gorm:"size:50;default:'environmental'"` // Тип события
	Difficulty       string    `json:"difficulty" gorm:"size:20;default:'easy'"`          // Сложность
	WeatherDependent bool      `json:"weather_dependent" gorm:"default:false"`            // Зависит от погоды
	Requirements     string    `json:"requirements" gorm:"type:text"`                     // Требования к участникам
	WhatToBring      string    `json:"what_to_bring" gorm:"type:text"`                    // Что взять с собой
	ContactInfo      string    `json:"contact_info" gorm:"type:text"`                     // Контактная информация
	IsActive         bool      `json:"is_active" gorm:"default:true"`
	IsPublic         bool      `json:"is_public" gorm:"default:true"`     // Публичное событие
	IsRecurring      bool      `json:"is_recurring" gorm:"default:false"` // Повторяющееся событие
	RecurringPattern string    `json:"recurring_pattern" gorm:"size:50"`  // Паттерн повторения
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Связи
	Creator      User               `json:"creator" gorm:"foreignKey:CreatorID"`
	Inventory    []EventInventory   `json:"inventory" gorm:"foreignKey:EventID"`
	Photos       []EventPhoto       `json:"photos" gorm:"foreignKey:EventID"`
	Participants []EventParticipant `json:"participants" gorm:"foreignKey:EventID"`
}

// EventInventory представляет связь между ивентом и инвентарем
type EventInventory struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	EventID     uint      `json:"event_id" gorm:"not null"`
	InventoryID uint      `json:"inventory_id" gorm:"not null"`
	QuantityMin int       `json:"quantity_min" gorm:"default:1"`
	QuantityMax int       `json:"quantity_max" gorm:"default:10"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Связи
	Event     Event     `json:"event" gorm:"foreignKey:EventID"`
	Inventory Inventory `json:"inventory" gorm:"foreignKey:InventoryID"`
}

// EventPhoto представляет фотографию ивента
type EventPhoto struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	EventID   uint      `json:"event_id" gorm:"not null"`
	FilePath  string    `json:"file_path" gorm:"not null;size:500"`
	IsBefore  bool      `json:"is_before" gorm:"default:true"` // true - до уборки, false - после
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Связи
	Event Event `json:"event" gorm:"foreignKey:EventID"`
}

// EventParticipant представляет участника ивента с расширенным функционалом
type EventParticipant struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	EventID   uint       `json:"event_id" gorm:"not null"`
	UserID    uint       `json:"user_id" gorm:"not null"`
	Status    string     `json:"status" gorm:"not null;default:'pending'"` // 'pending', 'accepted', 'rejected', 'joined', 'left', 'removed'
	JoinedAt  *time.Time `json:"joined_at"`
	LeftAt    *time.Time `json:"left_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Связи
	Event Event `json:"event" gorm:"foreignKey:EventID"`
	User  User  `json:"user" gorm:"foreignKey:UserID"`
}

// ParticipantInventory представляет инвентарь, который участник берет с собой
type ParticipantInventory struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	ParticipantID   uint      `json:"participant_id" gorm:"not null"`
	InventoryItemID *uint     `json:"inventory_item_id"` // nullable for custom items
	CustomName      string    `json:"custom_name"`       // для пользовательских предметов
	Quantity        int       `json:"quantity" gorm:"not null;default:1"`
	Brought         bool      `json:"brought" gorm:"default:false"` // принес ли участник предмет
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Связи
	Participant   EventParticipant `json:"participant" gorm:"foreignKey:ParticipantID"`
	InventoryItem *Inventory       `json:"inventory_item,omitempty" gorm:"foreignKey:InventoryItemID"`
}

// EventPhotoPost представляет фотографии после завершения ивента
type EventPhotoPost struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	EventID    uint      `json:"event_id" gorm:"not null"`
	UploaderID uint      `json:"uploader_id" gorm:"not null"`
	FilePath   string    `json:"file_path" gorm:"not null;size:500"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Связи
	Event    Event `json:"event" gorm:"foreignKey:EventID"`
	Uploader User  `json:"uploader" gorm:"foreignKey:UploaderID"`
}

// Rating представляет оценку участника другим участником
type Rating struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	EventID    uint      `json:"event_id" gorm:"not null"`
	FromUserID uint      `json:"from_user_id" gorm:"not null"`
	ToUserID   uint      `json:"to_user_id" gorm:"not null"`
	Score      int       `json:"score" gorm:"not null;check:score >= 1 AND score <= 10"`
	Comment    string    `json:"comment" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Связи
	Event    Event `json:"event" gorm:"foreignKey:EventID"`
	FromUser User  `json:"from_user" gorm:"foreignKey:FromUserID"`
	ToUser   User  `json:"to_user" gorm:"foreignKey:ToUserID"`
}

// UserRatingSummary представляет сводку рейтингов пользователя
type UserRatingSummary struct {
	UserID       uint      `json:"user_id" gorm:"primaryKey"`
	TotalScore   int       `json:"total_score" gorm:"not null;default:0"`
	VotesCount   int       `json:"votes_count" gorm:"not null;default:0"`
	AverageScore float64   `json:"average_score" gorm:"not null;default:0.0"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Связи
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// Complaint представляет жалобу на участника
type Complaint struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	EventID     uint       `json:"event_id" gorm:"not null"`
	FromUserID  uint       `json:"from_user_id" gorm:"not null"`
	AboutUserID uint       `json:"about_user_id" gorm:"not null"`
	ReasonCode  string     `json:"reason_code" gorm:"not null;size:50"`   // код причины жалобы
	ReasonText  string     `json:"reason_text" gorm:"type:text"`          // текст жалобы
	Status      string     `json:"status" gorm:"not null;default:'open'"` // 'open', 'under_review', 'resolved', 'dismissed'
	ResolvedAt  *time.Time `json:"resolved_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Связи
	Event     Event `json:"event" gorm:"foreignKey:EventID"`
	FromUser  User  `json:"from_user" gorm:"foreignKey:FromUserID"`
	AboutUser User  `json:"about_user" gorm:"foreignKey:AboutUserID"`
}

// Константы для статусов и причин жалоб
const (
	// Статусы участников
	ParticipantStatusPending  = "pending"
	ParticipantStatusAccepted = "accepted"
	ParticipantStatusRejected = "rejected"
	ParticipantStatusJoined   = "joined"
	ParticipantStatusLeft     = "left"
	ParticipantStatusRemoved  = "removed"

	// Статусы жалоб
	ComplaintStatusOpen        = "open"
	ComplaintStatusUnderReview = "under_review"
	ComplaintStatusResolved    = "resolved"
	ComplaintStatusDismissed   = "dismissed"

	// Коды причин жалоб
	ComplaintReasonInappropriate        = "inappropriate_behavior"
	ComplaintReasonNoShow               = "no_show"
	ComplaintReasonDisruptive           = "disruptive_behavior"
	ComplaintReasonInappropriateContent = "inappropriate_content"
	ComplaintReasonHarassment           = "harassment"
	ComplaintReasonOther                = "other"

	// Типы событий
	EventTypeEnvironmental = "environmental"
	EventTypeSocial        = "social"
	EventTypeEducational   = "educational"
	EventTypeCharity       = "charity"
	EventTypeSports        = "sports"
	EventTypeCulture       = "culture"
	EventTypeHealth        = "health"
	EventTypeTechnology    = "technology"

	// Сложность событий
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"

	// Паттерны повторения
	RecurringPatternDaily   = "daily"
	RecurringPatternWeekly  = "weekly"
	RecurringPatternMonthly = "monthly"
	RecurringPatternYearly  = "yearly"

	// Режимы вступления
	JoinModeFree     = "free"
	JoinModeApproval = "approval"
)

// GetComplaintReasons возвращает список доступных причин жалоб
func GetComplaintReasons() map[string]string {
	return map[string]string{
		ComplaintReasonInappropriate:        "Неподобающее поведение",
		ComplaintReasonNoShow:               "Не явился на мероприятие",
		ComplaintReasonDisruptive:           "Нарушение порядка",
		ComplaintReasonInappropriateContent: "Неподходящий контент",
		ComplaintReasonHarassment:           "Притеснение",
		ComplaintReasonOther:                "Другое",
	}
}

// GetEventTypes возвращает список доступных типов событий
func GetEventTypes() map[string]string {
	return map[string]string{
		EventTypeEnvironmental: "Экология",
		EventTypeSocial:        "Социальное",
		EventTypeEducational:   "Образование",
		EventTypeCharity:       "Благотворительность",
		EventTypeSports:        "Спорт",
		EventTypeCulture:       "Культура",
		EventTypeHealth:        "Здоровье",
		EventTypeTechnology:    "Технологии",
	}
}

// GetDifficultyLevels возвращает список доступных уровней сложности
func GetDifficultyLevels() map[string]string {
	return map[string]string{
		DifficultyEasy:   "Легкая",
		DifficultyMedium: "Средняя",
		DifficultyHard:   "Сложная",
	}
}

// GetRecurringPatterns возвращает список доступных паттернов повторения
func GetRecurringPatterns() map[string]string {
	return map[string]string{
		RecurringPatternDaily:   "Ежедневно",
		RecurringPatternWeekly:  "Еженедельно",
		RecurringPatternMonthly: "Ежемесячно",
		RecurringPatternYearly:  "Ежегодно",
	}
}

// GetJoinModes возвращает список доступных режимов вступления
func GetJoinModes() map[string]string {
	return map[string]string{
		JoinModeFree:     "Свободное",
		JoinModeApproval: "По одобрению",
	}
}

// BeforeCreate хук для установки времени создания
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для обновления времени изменения
func (e *Event) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для EventInventory
func (ei *EventInventory) BeforeCreate(tx *gorm.DB) error {
	ei.CreatedAt = time.Now()
	ei.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для EventInventory
func (ei *EventInventory) BeforeUpdate(tx *gorm.DB) error {
	ei.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для EventPhoto
func (ep *EventPhoto) BeforeCreate(tx *gorm.DB) error {
	ep.CreatedAt = time.Now()
	ep.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для EventPhoto
func (ep *EventPhoto) BeforeUpdate(tx *gorm.DB) error {
	ep.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для EventParticipant
func (ep *EventParticipant) BeforeCreate(tx *gorm.DB) error {
	ep.CreatedAt = time.Now()
	ep.UpdatedAt = time.Now()
	if ep.Status == "joined" {
		now := time.Now()
		ep.JoinedAt = &now
	}
	return nil
}

// BeforeUpdate хук для EventParticipant
func (ep *EventParticipant) BeforeUpdate(tx *gorm.DB) error {
	ep.UpdatedAt = time.Now()

	// Если статус изменился на "left", устанавливаем время выхода
	if ep.Status == "left" && ep.LeftAt == nil {
		now := time.Now()
		ep.LeftAt = &now
	}

	return nil
}

// BeforeCreate хук для ParticipantInventory
func (pi *ParticipantInventory) BeforeCreate(tx *gorm.DB) error {
	pi.CreatedAt = time.Now()
	pi.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для ParticipantInventory
func (pi *ParticipantInventory) BeforeUpdate(tx *gorm.DB) error {
	pi.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для EventPhotoPost
func (ep *EventPhotoPost) BeforeCreate(tx *gorm.DB) error {
	ep.CreatedAt = time.Now()
	ep.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для EventPhotoPost
func (ep *EventPhotoPost) BeforeUpdate(tx *gorm.DB) error {
	ep.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для Rating
func (r *Rating) BeforeCreate(tx *gorm.DB) error {
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для Rating
func (r *Rating) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для UserRatingSummary
func (urs *UserRatingSummary) BeforeCreate(tx *gorm.DB) error {
	urs.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для UserRatingSummary
func (urs *UserRatingSummary) BeforeUpdate(tx *gorm.DB) error {
	urs.UpdatedAt = time.Now()
	return nil
}

// BeforeCreate хук для Complaint
func (c *Complaint) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate хук для Complaint
func (c *Complaint) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()

	// Если статус изменился на "resolved" или "dismissed", устанавливаем время разрешения
	if (c.Status == "resolved" || c.Status == "dismissed") && c.ResolvedAt == nil {
		now := time.Now()
		c.ResolvedAt = &now
	}

	return nil
}
