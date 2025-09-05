package controllers

import (
	"strconv"

	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// BlockController обрабатывает HTTP запросы для блокировок
type BlockController struct {
	db *gorm.DB
}

// NewBlockController создает новый контроллер блокировок
func NewBlockController(db *gorm.DB) *BlockController {
	return &BlockController{db: db}
}

// BlockUser блокирует пользователя
func (c *BlockController) BlockUser(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	blockedUserID, err := strconv.ParseUint(ctx.Params("user_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if uint(blockedUserID) == userID {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Cannot block yourself",
		})
	}

	// Проверяем, что пользователь существует
	var blockedUser models.User
	err = c.db.First(&blockedUser, blockedUserID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get user",
		})
	}

	// Проверяем, не заблокирован ли уже пользователь
	var existingBlock models.Block
	err = c.db.Where("blocker_id = ? AND blocked_id = ?", userID, blockedUserID).First(&existingBlock).Error
	if err == nil {
		return ctx.Status(409).JSON(fiber.Map{
			"error": "User is already blocked",
		})
	}

	// Создаем блокировку
	block := models.Block{
		BlockerID: userID,
		BlockedID: uint(blockedUserID),
	}

	if err := c.db.Create(&block).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to block user",
		})
	}

	// Загружаем связанные данные
	c.db.Preload("Blocker").Preload("Blocked").First(&block, block.ID)

	return ctx.Status(201).JSON(fiber.Map{
		"block":   block,
		"message": "User blocked successfully",
	})
}

// UnblockUser разблокирует пользователя
func (c *BlockController) UnblockUser(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	blockedUserID, err := strconv.ParseUint(ctx.Params("user_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Ищем блокировку
	var block models.Block
	err = c.db.Where("blocker_id = ? AND blocked_id = ?", userID, blockedUserID).First(&block).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "User is not blocked",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get block",
		})
	}

	// Удаляем блокировку
	if err := c.db.Delete(&block).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to unblock user",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "User unblocked successfully",
	})
}

// GetBlockedUsers возвращает список заблокированных пользователей
func (c *BlockController) GetBlockedUsers(ctx *fiber.Ctx) error {
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

	var blocks []models.Block
	query := c.db.Preload("Blocked").Where("blocker_id = ?", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err = query.Find(&blocks).Error
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get blocked users",
		})
	}

	return ctx.JSON(fiber.Map{
		"blocks": blocks,
		"limit":  limit,
		"offset": offset,
	})
}

// IsBlocked проверяет, заблокирован ли пользователь
func (c *BlockController) IsBlocked(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	blockedUserID, err := strconv.ParseUint(ctx.Params("user_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	blocked, err := models.IsBlocked(c.db, userID, uint(blockedUserID))
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to check block status",
		})
	}

	return ctx.JSON(fiber.Map{
		"blocked": blocked,
	})
}

// GetBlockInfo возвращает информацию о блокировке
func (c *BlockController) GetBlockInfo(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	blockedUserID, err := strconv.ParseUint(ctx.Params("user_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var block models.Block
	err = c.db.Preload("Blocked").Where("blocker_id = ? AND blocked_id = ?", userID, blockedUserID).First(&block).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "User is not blocked",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get block info",
		})
	}

	return ctx.JSON(fiber.Map{
		"block": block,
	})
}

// GetBlockStats возвращает статистику блокировок
func (c *BlockController) GetBlockStats(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)

	// Количество заблокированных пользователей
	var blockedCount int64
	c.db.Model(&models.Block{}).Where("blocker_id = ?", userID).Count(&blockedCount)

	// Количество пользователей, которые заблокировали текущего пользователя
	var blockedByCount int64
	c.db.Model(&models.Block{}).Where("blocked_id = ?", userID).Count(&blockedByCount)

	// Последние блокировки
	var recentBlocks []models.Block
	c.db.Preload("Blocked").Where("blocker_id = ?", userID).
		Order("created_at DESC").Limit(5).Find(&recentBlocks)

	stats := map[string]interface{}{
		"blocked_count":    blockedCount,
		"blocked_by_count": blockedByCount,
		"recent_blocks":    recentBlocks,
	}

	return ctx.JSON(stats)
}
