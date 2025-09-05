package controllers

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"toloko-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AttachmentController обрабатывает HTTP запросы для вложений
type AttachmentController struct {
	db *gorm.DB
}

// NewAttachmentController создает новый контроллер вложений
func NewAttachmentController(db *gorm.DB) *AttachmentController {
	return &AttachmentController{db: db}
}

// UploadAttachment загружает вложение для сообщения
func (c *AttachmentController) UploadAttachment(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	messageID, err := strconv.ParseUint(ctx.Params("message_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	// Проверяем, что сообщение существует и пользователь имеет к нему доступ
	var message models.Message
	err = c.db.Where("id = ? AND (from_user_id = ? OR to_user_id = ?)",
		messageID, userID, userID).First(&message).Error
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

	// Получаем файл из формы
	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "No file provided",
		})
	}

	// Валидируем файл
	if err := c.validateFile(file); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Создаем папку для загрузок, если она не существует
	uploadDir := "uploads/messages"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	// Генерируем уникальное имя файла
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%d_%s", messageID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	// Сохраняем файл
	if err := ctx.SaveFile(file, filePath); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	// Создаем запись о вложении
	attachment := models.Attachment{
		MessageID:  uint(messageID),
		FilePath:   filePath,
		MimeType:   file.Header.Get("Content-Type"),
		Size:       file.Size,
		UploadedBy: userID,
	}

	if err := c.db.Create(&attachment).Error; err != nil {
		// Удаляем файл в случае ошибки
		os.Remove(filePath)
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to save attachment record",
		})
	}

	// Обновляем JSON с вложениями в сообщении
	attachmentsData, err := message.GetAttachmentsData()
	if err != nil {
		attachmentsData = []models.AttachmentData{}
	}

	attachmentsData = append(attachmentsData, models.AttachmentData{
		ID:       attachment.ID,
		FilePath: filePath,
		MimeType: attachment.MimeType,
		Size:     attachment.Size,
	})

	if err := message.SetAttachmentsData(attachmentsData); err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to update message attachments",
		})
	}

	c.db.Save(&message)

	return ctx.Status(201).JSON(fiber.Map{
		"attachment": attachment,
		"file_url":   fmt.Sprintf("/api/attachments/%d/file", attachment.ID),
	})
}

// GetAttachmentFile возвращает файл вложения
func (c *AttachmentController) GetAttachmentFile(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	attachmentID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	// Получаем вложение с проверкой доступа
	var attachment models.Attachment
	err = c.db.Preload("Message").Where("id = ?", attachmentID).First(&attachment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	// Проверяем, что пользователь имеет доступ к сообщению
	if attachment.Message.FromUserID != userID && attachment.Message.ToUserID != userID {
		return ctx.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Проверяем, что файл существует
	if _, err := os.Stat(attachment.FilePath); os.IsNotExist(err) {
		return ctx.Status(404).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	// Возвращаем файл
	return ctx.SendFile(attachment.FilePath)
}

// GetAttachmentInfo возвращает информацию о вложении
func (c *AttachmentController) GetAttachmentInfo(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	attachmentID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	// Получаем вложение с проверкой доступа
	var attachment models.Attachment
	err = c.db.Preload("Message").Preload("Uploader").Where("id = ?", attachmentID).First(&attachment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	// Проверяем, что пользователь имеет доступ к сообщению
	if attachment.Message.FromUserID != userID && attachment.Message.ToUserID != userID {
		return ctx.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	return ctx.JSON(fiber.Map{
		"attachment": attachment,
		"file_url":   fmt.Sprintf("/api/attachments/%d/file", attachment.ID),
	})
}

// DeleteAttachment удаляет вложение
func (c *AttachmentController) DeleteAttachment(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	attachmentID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid attachment ID",
		})
	}

	// Получаем вложение с проверкой доступа
	var attachment models.Attachment
	err = c.db.Preload("Message").Where("id = ?", attachmentID).First(&attachment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx.Status(404).JSON(fiber.Map{
				"error": "Attachment not found",
			})
		}
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get attachment",
		})
	}

	// Проверяем, что пользователь является загрузившим вложение
	if attachment.UploadedBy != userID {
		return ctx.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Удаляем файл
	if err := os.Remove(attachment.FilePath); err != nil {
		// Логируем ошибку, но продолжаем удаление записи
		fmt.Printf("Warning: Failed to delete file %s: %v\n", attachment.FilePath, err)
	}

	// Удаляем запись о вложении
	if err := c.db.Delete(&attachment).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to delete attachment record",
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Attachment deleted",
	})
}

// validateFile валидирует загружаемый файл
func (c *AttachmentController) validateFile(file *multipart.FileHeader) error {
	// Проверяем размер файла (10 MB)
	const maxSize = 10 * 1024 * 1024
	if file.Size > maxSize {
		return fmt.Errorf("file size exceeds 10MB limit")
	}

	// Проверяем тип файла
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return fmt.Errorf("file type not allowed. Only images are supported")
	}

	// Проверяем расширение файла
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

	allowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file extension not allowed")
	}

	// Проверяем имя файла на безопасность
	if strings.Contains(file.Filename, "..") || strings.Contains(file.Filename, "/") {
		return fmt.Errorf("invalid file name")
	}

	return nil
}

// GetMessageAttachments возвращает вложения сообщения
func (c *AttachmentController) GetMessageAttachments(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(uint)
	messageID, err := strconv.ParseUint(ctx.Params("message_id"), 10, 32)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid message ID",
		})
	}

	// Проверяем, что сообщение существует и пользователь имеет к нему доступ
	var message models.Message
	err = c.db.Where("id = ? AND (from_user_id = ? OR to_user_id = ?)",
		messageID, userID, userID).First(&message).Error
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

	// Получаем вложения
	var attachments []models.Attachment
	err = c.db.Where("message_id = ?", messageID).Find(&attachments).Error
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": "Failed to get attachments",
		})
	}

	// Добавляем URL для каждого вложения
	for i := range attachments {
		attachments[i].FilePath = fmt.Sprintf("/api/attachments/%d/file", attachments[i].ID)
	}

	return ctx.JSON(fiber.Map{
		"attachments": attachments,
	})
}
