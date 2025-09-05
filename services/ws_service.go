package services

import (
	"log"
	"sync"
	"time"

	"toloko-backend/models"

	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

// WSMessage представляет сообщение WebSocket
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	TempID  string      `json:"temp_id,omitempty"`
}

// MessagePayload представляет payload для отправки сообщения
type MessagePayload struct {
	Text        string   `json:"text"`
	Attachments []string `json:"attachments,omitempty"`
	ToUserID    uint     `json:"to_user_id"`
}

// TypingPayload представляет payload для индикатора набора текста
type TypingPayload struct {
	ConversationID uint `json:"conversation_id"`
	IsTyping       bool `json:"is_typing"`
}

// PresencePayload представляет payload для статуса присутствия
type PresencePayload struct {
	UserID   uint      `json:"user_id"`
	IsOnline bool      `json:"is_online"`
	LastSeen time.Time `json:"last_seen"`
}

// Client представляет подключенного клиента
type Client struct {
	ID       uint
	UserID   uint
	Conn     *websocket.Conn
	Send     chan WSMessage
	Hub      *Hub
	LastPing time.Time
}

// Hub управляет всеми подключениями
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan WSMessage
	mutex      sync.RWMutex
	db         *gorm.DB
}

// NewHub создает новый хаб
func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSMessage),
		db:         db,
	}
}

// Run запускает хаб
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

			// Обновляем статус присутствия
			models.UpdatePresence(h.db, client.UserID, true)

			// Уведомляем других пользователей о том, что пользователь онлайн
			h.broadcastPresenceUpdate(client.UserID, true)

			log.Printf("Client %d connected. Total clients: %d", client.UserID, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()

			// Обновляем статус присутствия
			models.UpdatePresence(h.db, client.UserID, false)

			// Уведомляем других пользователей о том, что пользователь офлайн
			h.broadcastPresenceUpdate(client.UserID, false)

			log.Printf("Client %d disconnected. Total clients: %d", client.UserID, len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// broadcastPresenceUpdate уведомляет о изменении статуса присутствия
func (h *Hub) broadcastPresenceUpdate(userID uint, isOnline bool) {
	presence := PresencePayload{
		UserID:   userID,
		IsOnline: isOnline,
		LastSeen: time.Now(),
	}

	message := WSMessage{
		Type:    "presence.update",
		Payload: presence,
	}

	h.mutex.RLock()
	for client := range h.clients {
		// Отправляем всем, кроме самого пользователя
		if client.UserID != userID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
	h.mutex.RUnlock()
}

// SendToUser отправляет сообщение конкретному пользователю
func (h *Hub) SendToUser(userID uint, message WSMessage) {
	h.mutex.RLock()
	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
	h.mutex.RUnlock()
}

// SendToConversation отправляет сообщение всем участникам диалога
func (h *Hub) SendToConversation(conversationID uint, message WSMessage, excludeUserID uint) {
	// Получаем участников диалога
	var conversation models.Conversation
	if err := h.db.Preload("UserA").Preload("UserB").First(&conversation, conversationID).Error; err != nil {
		log.Printf("Error getting conversation: %v", err)
		return
	}

	// Отправляем сообщение участникам диалога
	participants := []uint{conversation.UserAID, conversation.UserBID}

	h.mutex.RLock()
	for client := range h.clients {
		for _, participantID := range participants {
			if client.UserID == participantID && client.UserID != excludeUserID {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
	}
	h.mutex.RUnlock()
}

// HandleWebSocket обрабатывает WebSocket соединение
func (h *Hub) HandleWebSocket(c *websocket.Conn) {
	// Получаем JWT токен из query параметров
	tokenString := c.Query("token")
	if tokenString == "" {
		c.Close()
		return
	}

	// Парсим JWT токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("your-secret-key"), nil // TODO: вынести в конфиг
	})

	if err != nil || !token.Valid {
		c.Close()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.Close()
		return
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		c.Close()
		return
	}

	userID := uint(userIDFloat)

	// Создаем клиента
	client := &Client{
		ID:       uint(time.Now().UnixNano()),
		UserID:   userID,
		Conn:     c,
		Send:     make(chan WSMessage, 256),
		Hub:      h,
		LastPing: time.Now(),
	}

	// Регистрируем клиента
	h.register <- client

	// Запускаем горутины для чтения и записи
	go client.writePump()
	go client.readPump()
}

// readPump читает сообщения из WebSocket
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.LastPing = time.Now()
		return nil
	})

	for {
		var message WSMessage
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// writePump записывает сообщения в WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage обрабатывает входящие сообщения
func (c *Client) handleMessage(message WSMessage) {
	switch message.Type {
	case "message.send":
		c.handleSendMessage(message)
	case "message.read":
		c.handleReadMessage(message)
	case "typing.start":
		c.handleTypingStart(message)
	case "typing.stop":
		c.handleTypingStop(message)
	case "ping":
		c.handlePing(message)
	}
}

// handleSendMessage обрабатывает отправку сообщения
func (c *Client) handleSendMessage(message WSMessage) {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return
	}

	text, _ := payload["text"].(string)
	toUserIDFloat, _ := payload["to_user_id"].(float64)
	toUserID := uint(toUserIDFloat)

	// Проверяем, не заблокирован ли пользователь
	blocked, err := models.IsBlocked(c.Hub.db, c.UserID, toUserID)
	if err != nil || blocked {
		return
	}

	// Создаем или получаем диалог
	var conversation models.Conversation
	err = c.Hub.db.Where("(user_a_id = ? AND user_b_id = ?) OR (user_a_id = ? AND user_b_id = ?)",
		c.UserID, toUserID, toUserID, c.UserID).First(&conversation).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Создаем новый диалог
			conversation = models.Conversation{
				UserAID: c.UserID,
				UserBID: toUserID,
			}
			if err := c.Hub.db.Create(&conversation).Error; err != nil {
				log.Printf("Error creating conversation: %v", err)
				return
			}
		} else {
			log.Printf("Error getting conversation: %v", err)
			return
		}
	}

	// Создаем сообщение
	msg := models.Message{
		ConversationID: conversation.ID,
		FromUserID:     c.UserID,
		ToUserID:       toUserID,
		Text:           text,
		Status:         models.MessageStatusSent,
		TempID:         message.TempID,
	}

	if err := c.Hub.db.Create(&msg).Error; err != nil {
		log.Printf("Error creating message: %v", err)
		return
	}

	// Обновляем последнее сообщение в диалоге
	conversation.LastMessageID = &msg.ID
	c.Hub.db.Save(&conversation)

	// Отправляем подтверждение отправителю
	deliveryMessage := WSMessage{
		Type: "message.deliver",
		Payload: map[string]interface{}{
			"message_id": msg.ID,
			"temp_id":    message.TempID,
		},
	}
	c.Hub.SendToUser(c.UserID, deliveryMessage)

	// Отправляем сообщение получателю
	messagePayload := map[string]interface{}{
		"id":              msg.ID,
		"conversation_id": conversation.ID,
		"from_user_id":    msg.FromUserID,
		"text":            msg.Text,
		"status":          msg.Status,
		"created_at":      msg.CreatedAt,
	}

	receiveMessage := WSMessage{
		Type:    "message.receive",
		Payload: messagePayload,
	}
	c.Hub.SendToUser(toUserID, receiveMessage)
}

// handleReadMessage обрабатывает отметку о прочтении
func (c *Client) handleReadMessage(message WSMessage) {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return
	}

	messageIDFloat, _ := payload["message_id"].(float64)
	messageID := uint(messageIDFloat)

	// Обновляем статус сообщения
	var msg models.Message
	if err := c.Hub.db.First(&msg, messageID).Error; err != nil {
		return
	}

	// Проверяем, что сообщение адресовано текущему пользователю
	if msg.ToUserID != c.UserID {
		return
	}

	msg.Status = models.MessageStatusRead
	c.Hub.db.Save(&msg)

	// Уведомляем отправителя о прочтении
	readMessage := WSMessage{
		Type: "message.read",
		Payload: map[string]interface{}{
			"message_id": msg.ID,
		},
	}
	c.Hub.SendToUser(msg.FromUserID, readMessage)
}

// handleTypingStart обрабатывает начало набора текста
func (c *Client) handleTypingStart(message WSMessage) {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return
	}

	conversationIDFloat, _ := payload["conversation_id"].(float64)
	conversationID := uint(conversationIDFloat)

	typingMessage := WSMessage{
		Type: "typing.start",
		Payload: map[string]interface{}{
			"user_id":         c.UserID,
			"conversation_id": conversationID,
		},
	}

	c.Hub.SendToConversation(conversationID, typingMessage, c.UserID)
}

// handleTypingStop обрабатывает окончание набора текста
func (c *Client) handleTypingStop(message WSMessage) {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return
	}

	conversationIDFloat, _ := payload["conversation_id"].(float64)
	conversationID := uint(conversationIDFloat)

	typingMessage := WSMessage{
		Type: "typing.stop",
		Payload: map[string]interface{}{
			"user_id":         c.UserID,
			"conversation_id": conversationID,
		},
	}

	c.Hub.SendToConversation(conversationID, typingMessage, c.UserID)
}

// handlePing обрабатывает ping сообщения
func (c *Client) handlePing(message WSMessage) {
	pongMessage := WSMessage{
		Type: "pong",
		Payload: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}

	c.Hub.SendToUser(c.UserID, pongMessage)
}
