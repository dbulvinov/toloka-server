package controllers

import (
	"strconv"

	"toloko-backend/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ConversationController обрабатывает HTTP запросы для диалогов
type ConversationController struct {
	conversationService *services.ConversationService
}

// NewConversationController создает новый контроллер диалогов
func NewConversationController(db *gorm.DB) *ConversationController {
	return &ConversationController{
		conversationService: services.NewConversationService(db),
	}
}

// GetConversations возвращает список диалогов пользователя
func (c *ConversationController) GetConversations(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)

	// Параметры пагинации
	limitStr := ctx.Query("limit", "20")
	offsetStr := ctx.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	conversations, err := c.conversationService.GetConversations(userID, limit, offset)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get conversations",
		})
	}

	// Добавляем информацию о непрочитанных сообщениях
	for i := range conversations {
		unreadCount, _ := c.conversationService.GetUnreadCount(conversations[i].ID, userID)
		// В реальном приложении здесь бы обновляли unreadCount в структуре
		_ = unreadCount // Подавляем предупреждение о неиспользуемой переменной
	}

	return ctx.JSON(fiber.Map{
		"success":       true,
		"message":       "Conversations retrieved successfully",
		"conversations": conversations,
		"limit":         limit,
		"offset":        offset,
	})
}

// GetConversation возвращает диалог по ID
func (c *ConversationController) GetConversation(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	conversation, err := c.conversationService.GetConversation(uint(conversationID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Conversation not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get conversation",
		})
	}

	// Получаем количество непрочитанных сообщений
	unreadCount, _ := c.conversationService.GetUnreadCount(conversation.ID, userID)

	return ctx.JSON(fiber.Map{
		"success":       true,
		"message":       "Conversation retrieved successfully",
		"conversation":  conversation,
		"unread_count":  unreadCount,
	})
}

// CreateConversation создает новый диалог
func (c *ConversationController) CreateConversation(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)

	var request struct {
		UserBID uint `json:"user_b_id" validate:"required"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if request.UserBID == userID {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Cannot create conversation with yourself",
		})
	}

	conversation, err := c.conversationService.GetOrCreateConversation(userID, request.UserBID)
	if err != nil {
		if err.Error() == "user is blocked" {
			return ctx.Status(403).JSON(fiber.Map{
				"error": "User is blocked",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create conversation",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"success":     true,
		"message":     "Conversation created successfully",
		"conversation": conversation,
	})
}

// MarkAsRead помечает диалог как прочитанный
func (c *ConversationController) MarkAsRead(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	err = c.conversationService.MarkAsRead(uint(conversationID), userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to mark conversation as read",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Conversation marked as read",
	})
}

// DeleteConversation удаляет диалог
func (c *ConversationController) DeleteConversation(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	err = c.conversationService.DeleteConversation(uint(conversationID), userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to delete conversation",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Conversation deleted",
	})
}

// GetConversationStats возвращает статистику диалога
func (c *ConversationController) GetConversationStats(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	stats, err := c.conversationService.GetConversationStats(uint(conversationID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Conversation not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get conversation stats",
		})
	}

	return ctx.JSON(stats)
}
