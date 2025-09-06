package controllers

import (
	"strconv"

	"toloko-backend/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MessageController обрабатывает HTTP запросы для сообщений
type MessageController struct {
	messageService *services.MessageService
}

// NewMessageController создает новый контроллер сообщений
func NewMessageController(db *gorm.DB) *MessageController {
	return &MessageController{
		messageService: services.NewMessageService(db),
	}
}

// GetMessages возвращает сообщения диалога
func (c *MessageController) GetMessages(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("conversation_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	// Параметры пагинации
	limitStr := ctx.Query("limit", "50")
	beforeIDStr := ctx.Query("before")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	var beforeID *uint
	if beforeIDStr != "" {
		id, err := strconv.ParseUint(beforeIDStr, 10, 32)
		if err == nil {
			beforeID = &[]uint{uint(id)}[0]
		}
	}

	messages, err := c.messageService.GetMessages(uint(conversationID), userID, beforeID, limit)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get messages",
		})
	}

	return ctx.JSON(fiber.Map{
		"success":  true,
		"message":  "Messages retrieved successfully",
		"messages": messages,
		"limit":    limit,
		"before":   beforeID,
	})
}

// GetMessage возвращает сообщение по ID
func (c *MessageController) GetMessage(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	messageID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	message, err := c.messageService.GetMessage(uint(messageID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get message",
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "Message retrieved successfully",
		"data":    message,
	})
}

// SendMessage отправляет сообщение в диалог
func (c *MessageController) SendMessage(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("conversation_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	var request struct {
		ToUserID uint   `json:"to_user_id" validate:"required"`
		Text     string `json:"text" validate:"required"`
		TempID   string `json:"temp_id"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	message, err := c.messageService.CreateMessage(
		uint(conversationID),
		userID,
		request.ToUserID,
		request.Text,
		request.TempID,
	)
	if err != nil {
		if err.Error() == "user is blocked" {
			return ctx.Status(403).JSON(fiber.Map{
				"error": "User is blocked",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to send message",
		})
	}

	return ctx.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "Message sent successfully",
		"data":    message,
	})
}

// MarkAsRead помечает сообщение как прочитанное
func (c *MessageController) MarkAsRead(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	messageID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	err = c.messageService.MarkAsRead(uint(messageID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to mark message as read",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Message marked as read",
	})
}

// MarkConversationAsRead помечает все сообщения диалога как прочитанные
func (c *MessageController) MarkConversationAsRead(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	conversationID, err := strconv.ParseUint(ctx.Params("conversation_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	err = c.messageService.MarkConversationAsRead(uint(conversationID), userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to mark conversation as read",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Conversation marked as read",
	})
}

// DeleteMessage удаляет сообщение
func (c *MessageController) DeleteMessage(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	messageID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	err = c.messageService.DeleteMessage(uint(messageID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Message not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to delete message",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Message deleted",
	})
}

// SearchMessages ищет сообщения по тексту
func (c *MessageController) SearchMessages(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	query := ctx.Query("q")
	if query == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	limitStr := ctx.Query("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	messages, err := c.messageService.SearchMessages(userID, query, limit)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to search messages",
		})
	}

	return ctx.JSON(fiber.Map{
		"messages": messages,
		"query":    query,
		"limit":    limit,
	})
}

// GetUnreadMessages возвращает непрочитанные сообщения
func (c *MessageController) GetUnreadMessages(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)

	messages, err := c.messageService.GetUnreadMessages(userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get unread messages",
		})
	}

	return ctx.JSON(fiber.Map{
		"messages": messages,
	})
}

// GetMessageStats возвращает статистику сообщений
func (c *MessageController) GetMessageStats(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)

	stats, err := c.messageService.GetMessageStats(userID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get message stats",
		})
	}

	return ctx.JSON(stats)
}
