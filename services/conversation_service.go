package services

import (
	"errors"

	"toloko-backend/models"

	"gorm.io/gorm"
)

// ConversationService предоставляет методы для работы с диалогами
type ConversationService struct {
	db *gorm.DB
}

// NewConversationService создает новый сервис диалогов
func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{db: db}
}

// GetConversations возвращает список диалогов пользователя
func (s *ConversationService) GetConversations(userID uint, limit, offset int) ([]models.Conversation, error) {
	var conversations []models.Conversation

	query := s.db.Preload("UserA").Preload("UserB").Preload("LastMessage").
		Where("user_a_id = ? OR user_b_id = ?", userID, userID).
		Order("updated_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&conversations).Error
	return conversations, err
}

// GetConversation возвращает диалог по ID
func (s *ConversationService) GetConversation(conversationID, userID uint) (*models.Conversation, error) {
	var conversation models.Conversation

	err := s.db.Preload("UserA").Preload("UserB").Preload("LastMessage").
		Where("id = ? AND (user_a_id = ? OR user_b_id = ?)", conversationID, userID, userID).
		First(&conversation).Error

	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

// GetOrCreateConversation создает диалог или возвращает существующий
func (s *ConversationService) GetOrCreateConversation(userAID, userBID uint) (*models.Conversation, error) {
	// Проверяем, не заблокирован ли один пользователь другим
	blocked, err := models.IsBlocked(s.db, userAID, userBID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, errors.New("user is blocked")
	}

	// Ищем существующий диалог
	var conversation models.Conversation
	err = s.db.Where("(user_a_id = ? AND user_b_id = ?) OR (user_a_id = ? AND user_b_id = ?)",
		userAID, userBID, userBID, userAID).First(&conversation).Error

	if err == nil {
		// Диалог найден, загружаем связанные данные
		s.db.Preload("UserA").Preload("UserB").Preload("LastMessage").First(&conversation, conversation.ID)
		return &conversation, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Создаем новый диалог
	conversation = models.Conversation{
		UserAID: userAID,
		UserBID: userBID,
	}

	err = s.db.Create(&conversation).Error
	if err != nil {
		return nil, err
	}

	// Загружаем связанные данные
	s.db.Preload("UserA").Preload("UserB").First(&conversation, conversation.ID)

	return &conversation, nil
}

// GetConversationByUsers возвращает диалог между двумя пользователями
func (s *ConversationService) GetConversationByUsers(userAID, userBID uint) (*models.Conversation, error) {
	var conversation models.Conversation

	err := s.db.Where("(user_a_id = ? AND user_b_id = ?) OR (user_a_id = ? AND user_b_id = ?)",
		userAID, userBID, userBID, userAID).First(&conversation).Error

	if err != nil {
		return nil, err
	}

	// Загружаем связанные данные
	s.db.Preload("UserA").Preload("UserB").Preload("LastMessage").First(&conversation, conversation.ID)

	return &conversation, nil
}

// UpdateLastMessage обновляет последнее сообщение в диалоге
func (s *ConversationService) UpdateLastMessage(conversationID, messageID uint) error {
	return s.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Update("last_message_id", messageID).Error
}

// GetUnreadCount возвращает количество непрочитанных сообщений в диалоге
func (s *ConversationService) GetUnreadCount(conversationID, userID uint) (int64, error) {
	var count int64

	err := s.db.Model(&models.Message{}).
		Where("conversation_id = ? AND to_user_id = ? AND status != ?",
			conversationID, userID, models.MessageStatusRead).
		Count(&count).Error

	return count, err
}

// MarkAsRead помечает все сообщения в диалоге как прочитанные
func (s *ConversationService) MarkAsRead(conversationID, userID uint) error {
	return s.db.Model(&models.Message{}).
		Where("conversation_id = ? AND to_user_id = ? AND status != ?",
			conversationID, userID, models.MessageStatusRead).
		Update("status", models.MessageStatusRead).Error
}

// DeleteConversation удаляет диалог (мягкое удаление)
func (s *ConversationService) DeleteConversation(conversationID, userID uint) error {
	// Проверяем, что пользователь является участником диалога
	var conversation models.Conversation
	err := s.db.Where("id = ? AND (user_a_id = ? OR user_b_id = ?)",
		conversationID, userID, userID).First(&conversation).Error

	if err != nil {
		return err
	}

	// Удаляем диалог
	return s.db.Delete(&conversation).Error
}

// GetConversationStats возвращает статистику диалога
func (s *ConversationService) GetConversationStats(conversationID, userID uint) (map[string]interface{}, error) {
	// Проверяем, что пользователь является участником диалога
	var conversation models.Conversation
	err := s.db.Where("id = ? AND (user_a_id = ? OR user_b_id = ?)",
		conversationID, userID, userID).First(&conversation).Error

	if err != nil {
		return nil, err
	}

	// Получаем количество сообщений
	var totalMessages int64
	s.db.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&totalMessages)

	// Получаем количество непрочитанных сообщений
	var unreadMessages int64
	s.db.Model(&models.Message{}).
		Where("conversation_id = ? AND to_user_id = ? AND status != ?",
			conversationID, userID, models.MessageStatusRead).
		Count(&unreadMessages)

	// Получаем последнее сообщение
	var lastMessage models.Message
	s.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").First(&lastMessage)

	stats := map[string]interface{}{
		"conversation_id": conversationID,
		"total_messages":  totalMessages,
		"unread_messages": unreadMessages,
		"last_message_at": conversation.UpdatedAt,
		"created_at":      conversation.CreatedAt,
	}

	if lastMessage.ID != 0 {
		stats["last_message"] = lastMessage
	}

	return stats, nil
}
