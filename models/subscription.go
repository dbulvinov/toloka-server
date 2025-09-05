package models

import (
	"time"

	"gorm.io/gorm"
)

// Subscription представляет модель подписки пользователя на другого пользователя
type Subscription struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	SubscriberID   uint      `json:"subscriber_id" gorm:"not null"`
	SubscribedToID uint      `json:"subscribed_to_id" gorm:"not null"`
	CreatedAt      time.Time `json:"created_at"`

	// Связи
	Subscriber   User `json:"subscriber" gorm:"foreignKey:SubscriberID"`
	SubscribedTo User `json:"subscribed_to" gorm:"foreignKey:SubscribedToID"`
}

// BeforeCreate хук для установки времени создания
func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	s.CreatedAt = time.Now()
	return nil
}

// FeedEvent представляет событие в ленте с дополнительной информацией
type FeedEvent struct {
	Event
	IsFromSubscription bool `json:"is_from_subscription"`
}

// SubscriptionStats представляет статистику подписок пользователя
type SubscriptionStats struct {
	SubscriptionsCount int `json:"subscriptions_count"`
	SubscribersCount   int `json:"subscribers_count"`
}
