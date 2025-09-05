package services

import (
	"errors"
	"time"

	"toloko-backend/models"

	"gorm.io/gorm"
)

// MessageService предоставляет методы для работы с сообщениями
type MessageService struct {
	db *gorm.DB
}

// NewMessageService создает новый сервис сообщений
func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}

// GetMessages возвращает сообщения диалога с пагинацией
func (s *MessageService) GetMessages(conversationID, userID uint, beforeID *uint, limit int) ([]models.Message, error) {
	// Проверяем, что пользователь является участником диалога
	var conversation models.Conversation
	err := s.db.Where("id = ? AND (user_a_id = ? OR user_b_id = ?)",
		conversationID, userID, userID).First(&conversation).Error

	if err != nil {
		return nil, err
	}

	var messages []models.Message
	query := s.db.Preload("FromUser").Preload("ToUser").Preload("Attachments").
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC")

	if beforeID != nil {
		query = query.Where("id < ?", *beforeID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err = query.Find(&messages).Error
	return messages, err
}

// GetMessage возвращает сообщение по ID
func (s *MessageService) GetMessage(messageID, userID uint) (*models.Message, error) {
	var message models.Message

	err := s.db.Preload("FromUser").Preload("ToUser").Preload("Attachments").
		Where("id = ? AND (from_user_id = ? OR to_user_id = ?)", messageID, userID, userID).
		First(&message).Error

	if err != nil {
		return nil, err
	}

	return &message, nil
}

// CreateMessage создает новое сообщение
func (s *MessageService) CreateMessage(conversationID, fromUserID, toUserID uint, text, tempID string) (*models.Message, error) {
	// Проверяем, что пользователь является участником диалога
	var conversation models.Conversation
	err := s.db.Where("id = ? AND (user_a_id = ? OR user_b_id = ?)",
		conversationID, fromUserID, fromUserID).First(&conversation).Error

	if err != nil {
		return nil, err
	}

	// Проверяем, не заблокирован ли пользователь
	blocked, err := models.IsBlocked(s.db, fromUserID, toUserID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, errors.New("user is blocked")
	}

	message := models.Message{
		ConversationID: conversationID,
		FromUserID:     fromUserID,
		ToUserID:       toUserID,
		Text:           text,
		Status:         models.MessageStatusSent,
		TempID:         tempID,
	}

	err = s.db.Create(&message).Error
	if err != nil {
		return nil, err
	}

	// Обновляем последнее сообщение в диалоге
	conversation.LastMessageID = &message.ID
	s.db.Save(&conversation)

	// Загружаем связанные данные
	s.db.Preload("FromUser").Preload("ToUser").Preload("Attachments").First(&message, message.ID)

	return &message, nil
}

// UpdateMessageStatus обновляет статус сообщения
func (s *MessageService) UpdateMessageStatus(messageID uint, status models.MessageStatus) error {
	return s.db.Model(&models.Message{}).
		Where("id = ?", messageID).
		Update("status", status).Error
}

// MarkAsRead помечает сообщение как прочитанное
func (s *MessageService) MarkAsRead(messageID, userID uint) error {
	// Проверяем, что сообщение адресовано пользователю
	var message models.Message
	err := s.db.Where("id = ? AND to_user_id = ?", messageID, userID).First(&message).Error
	if err != nil {
		return err
	}

	message.Status = models.MessageStatusRead
	return s.db.Save(&message).Error
}

// MarkConversationAsRead помечает все сообщения диалога как прочитанные
func (s *MessageService) MarkConversationAsRead(conversationID, userID uint) error {
	// Проверяем, что пользователь является участником диалога
	var conversation models.Conversation
	err := s.db.Where("id = ? AND (user_a_id = ? OR user_b_id = ?)",
		conversationID, userID, userID).First(&conversation).Error

	if err != nil {
		return err
	}

	return s.db.Model(&models.Message{}).
		Where("conversation_id = ? AND to_user_id = ? AND status != ?",
			conversationID, userID, models.MessageStatusRead).
		Update("status", models.MessageStatusRead).Error
}

// DeleteMessage удаляет сообщение
func (s *MessageService) DeleteMessage(messageID, userID uint) error {
	// Проверяем, что пользователь является отправителем сообщения
	var message models.Message
	err := s.db.Where("id = ? AND from_user_id = ?", messageID, userID).First(&message).Error
	if err != nil {
		return err
	}

	// Удаляем вложения
	s.db.Where("message_id = ?", messageID).Delete(&models.Attachment{})

	// Удаляем сообщение
	return s.db.Delete(&message).Error
}

// SearchMessages ищет сообщения по тексту
func (s *MessageService) SearchMessages(userID uint, query string, limit int) ([]models.Message, error) {
	var messages []models.Message

	dbQuery := s.db.Preload("FromUser").Preload("ToUser").Preload("Attachments").
		Where("(from_user_id = ? OR to_user_id = ?) AND text LIKE ?",
			userID, userID, "%"+query+"%").
		Order("created_at DESC")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}

	err := dbQuery.Find(&messages).Error
	return messages, err
}

// GetUnreadMessages возвращает непрочитанные сообщения пользователя
func (s *MessageService) GetUnreadMessages(userID uint) ([]models.Message, error) {
	var messages []models.Message

	err := s.db.Preload("FromUser").Preload("ToUser").Preload("Attachments").
		Where("to_user_id = ? AND status != ?", userID, models.MessageStatusRead).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}

// GetMessageStats возвращает статистику сообщений
func (s *MessageService) GetMessageStats(userID uint) (map[string]interface{}, error) {
	// Общее количество сообщений
	var totalMessages int64
	s.db.Model(&models.Message{}).
		Where("from_user_id = ? OR to_user_id = ?", userID, userID).
		Count(&totalMessages)

	// Количество отправленных сообщений
	var sentMessages int64
	s.db.Model(&models.Message{}).
		Where("from_user_id = ?", userID).
		Count(&sentMessages)

	// Количество полученных сообщений
	var receivedMessages int64
	s.db.Model(&models.Message{}).
		Where("to_user_id = ?", userID).
		Count(&receivedMessages)

	// Количество непрочитанных сообщений
	var unreadMessages int64
	s.db.Model(&models.Message{}).
		Where("to_user_id = ? AND status != ?", userID, models.MessageStatusRead).
		Count(&unreadMessages)

	// Сообщения за последние 24 часа
	var recentMessages int64
	yesterday := time.Now().Add(-24 * time.Hour)
	s.db.Model(&models.Message{}).
		Where("(from_user_id = ? OR to_user_id = ?) AND created_at > ?",
			userID, userID, yesterday).
		Count(&recentMessages)

	stats := map[string]interface{}{
		"total_messages":    totalMessages,
		"sent_messages":     sentMessages,
		"received_messages": receivedMessages,
		"unread_messages":   unreadMessages,
		"recent_messages":   recentMessages,
	}

	return stats, nil
}

// GetMessagesByDateRange возвращает сообщения за определенный период
func (s *MessageService) GetMessagesByDateRange(userID uint, startDate, endDate time.Time) ([]models.Message, error) {
	var messages []models.Message

	err := s.db.Preload("FromUser").Preload("ToUser").Preload("Attachments").
		Where("(from_user_id = ? OR to_user_id = ?) AND created_at BETWEEN ? AND ?",
			userID, userID, startDate, endDate).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, err
}
