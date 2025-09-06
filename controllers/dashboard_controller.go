package controllers

import (
	"time"

	"toloko-backend/models"
	"toloko-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DashboardController контроллер для дашборда Home экрана
type DashboardController struct {
	db *gorm.DB
}

// NewDashboardController создает новый экземпляр DashboardController
func NewDashboardController(db *gorm.DB) *DashboardController {
	return &DashboardController{db: db}
}

// GetDashboardData получает все данные для дашборда Home экрана
func (dc *DashboardController) GetDashboardData(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := dc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Необходима авторизация",
		})
	}

	// Получаем данные пользователя
	var user models.User
	if err := dc.db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error":   true,
			"message": "Пользователь не найден",
		})
	}

	// Получаем уровень пользователя
	var userLevel models.UserLevel
	dc.db.Where("user_id = ?", userID).First(&userLevel)

	// Получаем статистику пользователя
	var userStats struct {
		EventsParticipated int64 `json:"events_participated"`
		EventsCreated      int64 `json:"events_created"`
		CommunitiesCreated int64 `json:"communities_created"`
		NewsCreated        int64 `json:"news_created"`
		CommentsCreated    int64 `json:"comments_created"`
		FollowersCount     int64 `json:"followers_count"`
		FollowingCount     int64 `json:"following_count"`
	}

	// Подсчитываем статистику
	dc.db.Model(&models.EventParticipant{}).Where("user_id = ?", userID).Count(&userStats.EventsParticipated)
	dc.db.Model(&models.Event{}).Where("created_by = ?", userID).Count(&userStats.EventsCreated)
	dc.db.Model(&models.Community{}).Where("created_by = ?", userID).Count(&userStats.CommunitiesCreated)
	dc.db.Model(&models.News{}).Where("created_by = ?", userID).Count(&userStats.NewsCreated)
	dc.db.Model(&models.Comment{}).Where("user_id = ?", userID).Count(&userStats.CommentsCreated)
	dc.db.Model(&models.Subscription{}).Where("target_user_id = ?", userID).Count(&userStats.FollowersCount)
	dc.db.Model(&models.Subscription{}).Where("user_id = ?", userID).Count(&userStats.FollowingCount)

	// Получаем предстоящие события (следующие 7 дней)
	var upcomingEvents []models.Event
	now := time.Now()
	weekFromNow := now.AddDate(0, 0, 7)
	dc.db.Preload("Creator").
		Preload("Photos").
		Preload("Inventory.Inventory").
		Preload("Participants.User").
		Where("start_time > ? AND start_time <= ? AND is_active = ?", now, weekFromNow, true).
		Order("start_time ASC").
		Limit(5).
		Find(&upcomingEvents)

	// Получаем рекомендуемые события (от других пользователей)
	var recommendedEvents []models.Event
	dc.db.Preload("Creator").
		Preload("Photos").
		Preload("Inventory.Inventory").
		Preload("Participants.User").
		Where("creator_id != ? AND start_time > ? AND is_active = ?", userID, now, true).
		Order("start_time ASC").
		Limit(3).
		Find(&recommendedEvents)

	// Получаем последние достижения пользователя
	var recentAchievements []models.UserAchievement
	dc.db.Preload("Achievement").
		Where("user_id = ?", userID).
		Order("earned_at DESC").
		Limit(3).
		Find(&recentAchievements)

	// Получаем топ-5 пользователей для лидерборда
	var leaderboard []models.UserLevel
	dc.db.Preload("User").
		Order("points DESC, level DESC").
		Limit(5).
		Find(&leaderboard)

	// Получаем статистику сообщества
	var communityStats struct {
		TotalEvents      int64 `json:"total_events"`
		ActiveEvents     int64 `json:"active_events"`
		TotalUsers       int64 `json:"total_users"`
		TotalCommunities int64 `json:"total_communities"`
	}

	dc.db.Model(&models.Event{}).Where("is_active = ?", true).Count(&communityStats.TotalEvents)
	dc.db.Model(&models.Event{}).Where("start_time <= ? AND end_time >= ? AND is_active = ?", now, now, true).Count(&communityStats.ActiveEvents)
	dc.db.Model(&models.User{}).Where("is_active = ?", true).Count(&communityStats.TotalUsers)
	dc.db.Model(&models.Community{}).Where("is_active = ?", true).Count(&communityStats.TotalCommunities)

	// Получаем события на сегодня
	var todayEvents []models.Event
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)
	dc.db.Preload("Creator").
		Preload("Photos").
		Where("start_time >= ? AND start_time < ? AND is_active = ?", startOfDay, endOfDay, true).
		Order("start_time ASC").
		Find(&todayEvents)

	// Формируем ответ
	response := fiber.Map{
		"user": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"avatar":     user.Avatar,
			"level":      userLevel.Level,
			"points":     userLevel.Points,
			"max_points": userLevel.Level * 100,
			"current_xp": userLevel.Points - ((userLevel.Level - 1) * 100),
			"max_xp":     100,
		},
		"stats":               userStats,
		"upcoming_events":     upcomingEvents,
		"recommended_events":  recommendedEvents,
		"recent_achievements": recentAchievements,
		"leaderboard":         leaderboard,
		"community_stats":     communityStats,
		"today_events":        todayEvents,
		"error":               false,
		"message":             "Данные дашборда получены успешно",
	}

	return c.JSON(response)
}

// GetUserDashboard получает упрощенные данные дашборда для текущего пользователя
func (dc *DashboardController) GetUserDashboard(c *fiber.Ctx) error {
	// Получаем ID пользователя из JWT токена
	userID, err := dc.getUserIDFromToken(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Необходима авторизация",
		})
	}

	// Получаем данные пользователя
	var user models.User
	if err := dc.db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error":   true,
			"message": "Пользователь не найден",
		})
	}

	// Получаем уровень пользователя
	var userLevel models.UserLevel
	dc.db.Where("user_id = ?", userID).First(&userLevel)

	// Получаем статистику пользователя
	var userStats struct {
		EventsParticipated int64 `json:"events_participated"`
		EventsCreated      int64 `json:"events_created"`
		CommunitiesCreated int64 `json:"communities_created"`
		FollowersCount     int64 `json:"followers_count"`
		FollowingCount     int64 `json:"following_count"`
	}

	// Подсчитываем статистику
	dc.db.Model(&models.EventParticipant{}).Where("user_id = ?", userID).Count(&userStats.EventsParticipated)
	dc.db.Model(&models.Event{}).Where("created_by = ?", userID).Count(&userStats.EventsCreated)
	dc.db.Model(&models.Community{}).Where("created_by = ?", userID).Count(&userStats.CommunitiesCreated)
	dc.db.Model(&models.Subscription{}).Where("target_user_id = ?", userID).Count(&userStats.FollowersCount)
	dc.db.Model(&models.Subscription{}).Where("user_id = ?", userID).Count(&userStats.FollowingCount)

	// Получаем предстоящие события
	var upcomingEvents []models.Event
	now := time.Now()
	dc.db.Preload("Creator").
		Preload("Photos").
		Where("start_time > ? AND is_active = ?", now, true).
		Order("start_time ASC").
		Limit(3).
		Find(&upcomingEvents)

	// Получаем статистику ленты
	var feedStats struct {
		SubscriptionsCount int64 `json:"subscriptions_count"`
		UpcomingEvents     int64 `json:"upcoming_events"`
		TodayEvents        int64 `json:"today_events"`
	}

	dc.db.Model(&models.Subscription{}).Where("subscriber_id = ?", userID).Count(&feedStats.SubscriptionsCount)
	dc.db.Model(&models.Event{}).Where("start_time > ? AND is_active = ?", now, true).Count(&feedStats.UpcomingEvents)

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)
	dc.db.Model(&models.Event{}).Where("start_time >= ? AND start_time < ? AND is_active = ?", startOfDay, endOfDay, true).Count(&feedStats.TodayEvents)

	// Формируем ответ
	response := fiber.Map{
		"user": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"avatar":     user.Avatar,
			"level":      userLevel.Level,
			"points":     userLevel.Points,
			"max_points": userLevel.Level * 100,
			"current_xp": userLevel.Points - ((userLevel.Level - 1) * 100),
			"max_xp":     100,
		},
		"stats":           userStats,
		"upcoming_events": upcomingEvents,
		"feed_stats":      feedStats,
		"error":           false,
		"message":         "Данные дашборда получены успешно",
	}

	return c.JSON(response)
}

// getUserIDFromToken извлекает ID пользователя из JWT токена
func (dc *DashboardController) getUserIDFromToken(c *fiber.Ctx) (uint, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0, fiber.NewError(401, "Отсутствует токен авторизации")
	}

	// Извлекаем токен из заголовка "Bearer <token>"
	token := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	// Валидируем токен
	claims, err := utils.ValidateJWT(token)
	if err != nil {
		return 0, fiber.NewError(401, "Недействительный токен")
	}

	return claims.UserID, nil
}
