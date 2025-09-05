package main

import (
	"log"
	"os"
	"time"

	"toloko-backend/controllers"
	"toloko-backend/models"
	"toloko-backend/routes"
	"toloko-backend/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

func main() {
	// Инициализация базы данных
	db, err := models.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Автомиграция
	db.AutoMigrate(&models.User{}, &models.Event{}, &models.Inventory{}, &models.EventInventory{}, &models.EventPhoto{}, &models.EventParticipant{}, &models.ParticipantInventory{}, &models.EventPhotoPost{}, &models.Rating{}, &models.UserRatingSummary{}, &models.Complaint{}, &models.Subscription{}, &models.Community{}, &models.CommunityRole{}, &models.News{}, &models.Comment{}, &models.NewsLike{}, &models.Achievement{}, &models.UserAchievement{}, &models.UserLevel{}, &models.PinnedPost{}, &models.Conversation{}, &models.Message{}, &models.Attachment{}, &models.Block{}, &models.UserPresence{})

	// Инициализация базового инвентаря
	initDefaultInventory(db)

	// Инициализация базовых достижений
	initDefaultAchievements(db)

	// Создание Fiber приложения
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
				"code":    code,
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	
	// CORS настройки
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000,http://127.0.0.1:3000,http://10.0.2.2:8080,http://192.168.0.191:8080"
	}
	
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Инициализация контроллеров
	authController := controllers.NewAuthController(db)
	eventController := controllers.NewEventController(db)
	subscriptionController := controllers.NewSubscriptionController(db)
	feedController := controllers.NewFeedController(db)
	communityController := controllers.NewCommunityController(db)
	newsController := controllers.NewNewsController(db)
	commentController := controllers.NewCommentController(db)
	userController := controllers.NewUserController(db)
	achievementController := controllers.NewAchievementController(db)
	levelController := controllers.NewLevelController(db)
	participantController := controllers.NewParticipantController(db)
	ratingController := controllers.NewRatingController(db)
	complaintController := controllers.NewComplaintController(db)

	// Настройка маршрутов
	routes.SetupAuthRoutes(app, authController)
	routes.SetupEventRoutes(app, eventController)
	routes.SetupSubscriptionRoutes(app, subscriptionController)
	routes.SetupFeedRoutes(app, feedController)
	routes.SetupCommunityRoutes(app, communityController)
	routes.SetupNewsRoutes(app, newsController)
	routes.SetupCommentRoutes(app, commentController)
	routes.SetupUserRoutes(app, userController)
	routes.SetupAchievementRoutes(app, achievementController)
	routes.SetupLevelRoutes(app, levelController)
	routes.SetupParticipantRoutes(app, participantController)
	routes.SetupRatingRoutes(app, ratingController)
	routes.SetupComplaintRoutes(app, complaintController)

	// Настройка маршрутов для модуля сообщений
	routes.SetupConversationRoutes(app, db)
	routes.SetupMessageRoutes(app, db)
	routes.SetupAttachmentRoutes(app, db)
	routes.SetupBlockRoutes(app, db)

	// Инициализация WebSocket хаба
	hub := services.NewHub(db)
	go hub.Run()

	// WebSocket маршрут
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		hub.HandleWebSocket(c)
	}))

	// Общий health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"message":   "Toloko Backend is running",
			"timestamp": time.Now().Unix(),
		})
	})

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// initDefaultInventory инициализирует базовый инвентарь в системе
func initDefaultInventory(db *gorm.DB) {
	// Список базового инвентаря для волонтерских мероприятий
	defaultInventory := []models.Inventory{
		{Name: "Перчатки рабочие", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Мешки для мусора", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Грабли", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Лопата", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Ведро", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Веник", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Совок", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Щетка", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Средство для мытья", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Тряпки", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Аптечка", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Вода питьевая", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Снеки", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Куртка защитная", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Сапоги резиновые", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Фонарик", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Радио", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Знаки безопасности", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Конусы", IsCustom: false, CreatedBy: 0, IsActive: true},
		{Name: "Лента оградительная", IsCustom: false, CreatedBy: 0, IsActive: true},
	}

	// Проверяем, есть ли уже инвентарь в базе
	var count int64
	db.Model(&models.Inventory{}).Where("is_custom = ?", false).Count(&count)

	if count == 0 {
		log.Println("Инициализация базового инвентаря...")
		for _, item := range defaultInventory {
			if err := db.Create(&item).Error; err != nil {
				log.Printf("Ошибка при создании инвентаря '%s': %v", item.Name, err)
			} else {
				log.Printf("Создан инвентарь: %s", item.Name)
			}
		}
		log.Println("Базовый инвентарь инициализирован")
	} else {
		log.Printf("Базовый инвентарь уже существует (%d элементов)", count)
	}
}

// initDefaultAchievements инициализирует базовые достижения в системе
func initDefaultAchievements(db *gorm.DB) {
	// Список базовых достижений
	defaultAchievements := []models.Achievement{
		{Name: "Первые шаги", Description: "Создайте аккаунт в системе", IconPath: "/icons/first-steps.png", Points: 10, Category: "registration", IsActive: true},
		{Name: "Волонтер-новичок", Description: "Участвуйте в первом мероприятии", IconPath: "/icons/volunteer-beginner.png", Points: 25, Category: "participation", IsActive: true},
		{Name: "Активный участник", Description: "Участвуйте в 5 мероприятиях", IconPath: "/icons/active-participant.png", Points: 50, Category: "participation", IsActive: true},
		{Name: "Опытный волонтер", Description: "Участвуйте в 10 мероприятиях", IconPath: "/icons/experienced-volunteer.png", Points: 100, Category: "participation", IsActive: true},
		{Name: "Лидер мероприятий", Description: "Участвуйте в 25 мероприятиях", IconPath: "/icons/event-leader.png", Points: 250, Category: "participation", IsActive: true},
		{Name: "Организатор", Description: "Создайте первое мероприятие", IconPath: "/icons/organizer.png", Points: 75, Category: "organization", IsActive: true},
		{Name: "Активный организатор", Description: "Создайте 5 мероприятий", IconPath: "/icons/active-organizer.png", Points: 150, Category: "organization", IsActive: true},
		{Name: "Создатель сообщества", Description: "Создайте первое сообщество", IconPath: "/icons/community-creator.png", Points: 100, Category: "community", IsActive: true},
		{Name: "Активный автор", Description: "Создайте 10 новостей", IconPath: "/icons/active-author.png", Points: 50, Category: "content", IsActive: true},
		{Name: "Комментатор", Description: "Оставьте 50 комментариев", IconPath: "/icons/commentator.png", Points: 25, Category: "content", IsActive: true},
		{Name: "Социальный активист", Description: "Подпишитесь на 10 пользователей", IconPath: "/icons/social-activist.png", Points: 30, Category: "social", IsActive: true},
		{Name: "Популярный пользователь", Description: "Получите 10 подписчиков", IconPath: "/icons/popular-user.png", Points: 40, Category: "social", IsActive: true},
		{Name: "Эксперт по уборке", Description: "Участвуйте в 5 уборках", IconPath: "/icons/cleaning-expert.png", Points: 75, Category: "specialization", IsActive: true},
		{Name: "Экологический активист", Description: "Участвуйте в 3 экологических мероприятиях", IconPath: "/icons/eco-activist.png", Points: 60, Category: "specialization", IsActive: true},
		{Name: "Помощник животных", Description: "Участвуйте в 3 мероприятиях помощи животным", IconPath: "/icons/animal-helper.png", Points: 60, Category: "specialization", IsActive: true},
	}

	// Проверяем, есть ли уже достижения в базе
	var count int64
	db.Model(&models.Achievement{}).Where("is_active = ?", true).Count(&count)

	if count == 0 {
		log.Println("Инициализация базовых достижений...")
		for _, achievement := range defaultAchievements {
			if err := db.Create(&achievement).Error; err != nil {
				log.Printf("Ошибка при создании достижения '%s': %v", achievement.Name, err)
			} else {
				log.Printf("Создано достижение: %s", achievement.Name)
			}
		}
		log.Println("Базовые достижения инициализированы")
	} else {
		log.Printf("Базовые достижения уже существуют (%d элементов)", count)
	}
}
