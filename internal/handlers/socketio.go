package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"im-demo/internal/config"
	"im-demo/internal/models"
	"im-demo/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zishang520/socket.io/v2/socket"
)

// SocketIOHandler handles Socket.IO connections and events using v4+ protocol
type SocketIOHandler struct {
	server       *socket.Server
	redisService *services.RedisService
	config       *config.Config
	logger       *logrus.Logger
	sessions     map[string]*models.User // session_id -> user
	userSessions map[string][]string     // username -> []session_ids (支持多设备)
}

// NewSocketIOHandler creates a new Socket.IO handler with v4+ protocol support
func NewSocketIOHandler(cfg *config.Config, redisService *services.RedisService, logger *logrus.Logger) (*SocketIOHandler, error) {
	// Create server with v4+ protocol support
	server := socket.NewServer(nil, nil)

	handler := &SocketIOHandler{
		server:       server,
		redisService: redisService,
		config:       cfg,
		logger:       logger,
		sessions:     make(map[string]*models.User),
		userSessions: make(map[string][]string), // 新增：用户名到会话列表的映射
	}

	// Setup event handlers
	handler.setupEventHandlers()

	// Setup Redis subscription for distributed messaging
	go handler.subscribeToRedis()

	return handler, nil
}

// setupEventHandlers sets up Socket.IO event handlers using v4+ API
func (h *SocketIOHandler) setupEventHandlers() {
	// Connection event
	h.server.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		sessionID := string(client.Id())

		h.logger.WithField("session_id", sessionID).Info("New connection established")

		// 为每个连接创建私有房间，用于点对点消息
		client.Join(socket.Room(sessionID))

		// User join event - 支持多设备登录
		client.On("join", func(args ...any) {
			if len(args) == 0 {
				h.sendError(client, "No user data provided")
				return
			}

			data, ok := args[0].(map[string]interface{})
			if !ok {
				h.sendError(client, "Invalid user data")
				return
			}

			userName, _ := data["userName"].(string)
			deviceInfo, _ := data["deviceInfo"].(string) // 新增：设备信息
			avatar, _ := data["avatar"].(string)

			if userName == "" {
				h.sendError(client, "User name is required")
				return
			}

			// 使用用户名作为唯一标识，支持多设备
			if deviceInfo == "" {
				deviceInfo = "Unknown Device"
			}

			user := &models.User{
				ID:       userName, // 使用用户名作为用户ID
				Name:     userName,
				Avatar:   avatar,
				Status:   "online",
				LastSeen: time.Now(),
				Metadata: map[string]interface{}{
					"deviceInfo": deviceInfo,
					"sessionId":  sessionID,
				},
			}

			// 存储会话信息
			h.sessions[sessionID] = user

			// 添加到用户的会话列表
			if h.userSessions[userName] == nil {
				h.userSessions[userName] = []string{}
			}
			h.userSessions[userName] = append(h.userSessions[userName], sessionID)

			// 在Redis中存储用户会话信息
			ctx := context.Background()
			h.redisService.StoreUserSession(ctx, userName, sessionID)

			// 广播用户上线（如果是该用户的第一个设备）
			if len(h.userSessions[userName]) == 1 {
				h.broadcastUserStatus(userName, "online")
			}

			// 发送确认消息，包含设备信息
			client.Emit("joined", map[string]interface{}{
				"userId":      userName,
				"userName":    userName,
				"deviceInfo":  deviceInfo,
				"status":      "online",
				"deviceCount": len(h.userSessions[userName]), // 当前设备数量
			})

			// 向用户的其他设备广播新设备登录
			h.broadcastToUserDevices(userName, "device_connected", map[string]interface{}{
				"deviceInfo":  deviceInfo,
				"sessionId":   sessionID,
				"deviceCount": len(h.userSessions[userName]),
			}, sessionID) // 排除当前会话

			h.logger.WithFields(logrus.Fields{
				"user_name":    userName,
				"session_id":   sessionID,
				"device_info":  deviceInfo,
				"device_count": len(h.userSessions[userName]),
			}).Info("User joined with device")
		})

		// Join room event
		client.On("join_room", func(args ...any) {
			if len(args) == 0 {
				h.sendError(client, "No room data provided")
				return
			}

			data, ok := args[0].(map[string]interface{})
			if !ok {
				h.sendError(client, "Invalid room data")
				return
			}

			roomID, _ := data["roomId"].(string)
			userName, _ := data["userName"].(string) // 改为userName

			if roomID == "" || userName == "" {
				h.sendError(client, "Invalid room or user data")
				return
			}

			// Join the room
			client.Join(socket.Room(roomID))

			// Add user to room in Redis
			ctx := context.Background()
			h.redisService.AddUserToRoom(ctx, roomID, userName)

			// Broadcast to room
			h.server.To(socket.Room(roomID)).Emit("user_joined_room", map[string]interface{}{
				"userName": userName,
				"roomId":   roomID,
			})

			// Send confirmation
			client.Emit("room_joined", map[string]interface{}{
				"roomId":   roomID,
				"userName": userName,
			})

			h.logger.WithFields(logrus.Fields{
				"user_name":  userName,
				"room_id":    roomID,
				"session_id": sessionID,
			}).Info("User joined room")
		})

		// Leave room event
		client.On("leave_room", func(args ...any) {
			if len(args) == 0 {
				h.sendError(client, "No room data provided")
				return
			}

			data, ok := args[0].(map[string]interface{})
			if !ok {
				h.sendError(client, "Invalid room data")
				return
			}

			roomID, _ := data["roomId"].(string)
			userName, _ := data["userName"].(string) // 改为userName

			if roomID == "" || userName == "" {
				h.sendError(client, "Invalid room or user data")
				return
			}

			// Leave the room
			client.Leave(socket.Room(roomID))

			// Remove user from room in Redis
			ctx := context.Background()
			h.redisService.RemoveUserFromRoom(ctx, roomID, userName)

			// Broadcast to room
			h.server.To(socket.Room(roomID)).Emit("user_left_room", map[string]interface{}{
				"userName": userName,
				"roomId":   roomID,
			})

			h.logger.WithFields(logrus.Fields{
				"user_name":  userName,
				"room_id":    roomID,
				"session_id": sessionID,
			}).Info("User left room")
		})

		// Message event
		client.On("message", func(args ...any) {
			h.handleMessage(client, args...)
		})

		// File upload event
		client.On("file_upload", func(args ...any) {
			h.handleFileUpload(client, args...)
		})

		// Typing event
		client.On("typing", func(args ...any) {
			if len(args) == 0 {
				return
			}

			data, ok := args[0].(map[string]interface{})
			if !ok {
				return
			}

			roomID, _ := data["roomId"].(string)
			userName, _ := data["userName"].(string) // 改为userName

			if roomID != "" && userName != "" {
				h.server.To(socket.Room(roomID)).Emit("typing", map[string]interface{}{
					"userName": userName,
					"roomId":   roomID,
				})
			}
		})

		// Stop typing event
		client.On("stop_typing", func(args ...any) {
			if len(args) == 0 {
				return
			}

			data, ok := args[0].(map[string]interface{})
			if !ok {
				return
			}

			roomID, _ := data["roomId"].(string)
			userName, _ := data["userName"].(string) // 改为userName

			if roomID != "" && userName != "" {
				h.server.To(socket.Room(roomID)).Emit("stop_typing", map[string]interface{}{
					"userName": userName,
					"roomId":   roomID,
				})
			}
		})

		// Disconnect event - 支持多设备登录
		client.On("disconnect", func(args ...any) {
			reason := "unknown"
			if len(args) > 0 {
				if r, ok := args[0].(string); ok {
					reason = r
				}
			}

			h.logger.WithFields(logrus.Fields{
				"session_id": sessionID,
				"reason":     reason,
			}).Info("Device disconnected")

			// 清理用户会话
			if user, exists := h.sessions[sessionID]; exists {
				userName := user.ID

				// 从用户会话列表中移除当前会话
				if sessions, ok := h.userSessions[userName]; ok {
					for i, sid := range sessions {
						if sid == sessionID {
							h.userSessions[userName] = append(sessions[:i], sessions[i+1:]...)
							break
						}
					}

					// 如果用户的所有设备都下线了，广播用户离线
					if len(h.userSessions[userName]) == 0 {
						delete(h.userSessions, userName)
						ctx := context.Background()
						h.redisService.DeleteUserSession(ctx, userName)
						h.broadcastUserStatus(userName, "offline")
					} else {
						// 向用户的其他设备广播设备断开
						h.broadcastToUserDevices(userName, "device_disconnected", map[string]interface{}{
							"sessionId":   sessionID,
							"deviceCount": len(h.userSessions[userName]),
						}, "")
					}
				}

				delete(h.sessions, sessionID)
			}
		})
	})
}

// broadcastToUserDevices 向指定用户的所有设备广播消息
func (h *SocketIOHandler) broadcastToUserDevices(userName, event string, data map[string]interface{}, excludeSessionID string) {
	if sessions, ok := h.userSessions[userName]; ok {
		for _, sessionID := range sessions {
			if excludeSessionID != "" && sessionID == excludeSessionID {
				continue // 排除指定的会话
			}

			// 通过session ID向指定socket发送消息
			h.server.To(socket.Room(sessionID)).Emit(event, data)
		}
	}
}

// handleMessage handles incoming messages using v4+ protocol
func (h *SocketIOHandler) handleMessage(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		h.sendError(client, "No message data")
		return
	}

	data, ok := args[0].(map[string]interface{})
	if !ok {
		h.sendError(client, "Invalid message data")
		return
	}

	messageType, _ := data["type"].(string)
	content, _ := data["content"].(string)
	sender, _ := data["sender"].(string)
	roomID, _ := data["roomId"].(string)
	receiver, _ := data["receiver"].(string)

	if content == "" || sender == "" {
		h.sendError(client, "Invalid message data")
		return
	}
	for i := 0; i < 10; i++ {
		// Create message
		message := &models.Message{
			ID:        generateMessageID(),
			Type:      models.MessageType(messageType),
			Content:   content,
			Sender:    sender,
			Room:      roomID,
			Receiver:  receiver,
			Timestamp: time.Now(),
		}

		// Store message in Redis
		ctx := context.Background()
		if err := h.redisService.StoreMessage(ctx, message); err != nil {
			h.logger.WithError(err).Error("Failed to store message")
			h.sendError(client, "Failed to send message")
			return
		}

		// Broadcast message

		message.Content = message.Content + fmt.Sprintf("Message %d", i)
		h.broadcastMessage(message)

		h.logger.WithFields(logrus.Fields{
			"message_id": message.ID,
			"sender":     sender,
			"room_id":    roomID,
			"type":       messageType,
		}).Info("Message sent")
	}
}

// handleFileUpload handles file uploads using v4+ protocol
func (h *SocketIOHandler) handleFileUpload(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		h.sendError(client, "No file data")
		return
	}

	data, ok := args[0].(map[string]interface{})
	if !ok {
		h.sendError(client, "Invalid file data")
		return
	}

	fileName, _ := data["fileName"].(string)
	fileData, _ := data["fileData"].(string)
	fileType, _ := data["fileType"].(string)
	sender, _ := data["sender"].(string)
	roomID, _ := data["roomId"].(string)

	if fileName == "" || fileData == "" || sender == "" {
		h.sendError(client, "Invalid file data")
		return
	}

	// Decode base64 file data
	decodedData, err := base64.StdEncoding.DecodeString(fileData)
	if err != nil {
		h.logger.WithError(err).Error("Failed to decode file data")
		h.sendError(client, "Invalid file data")
		return
	}

	// Check file size
	if int64(len(decodedData)) > h.config.Upload.MaxFileSize {
		h.sendError(client, fmt.Sprintf("File too large, max size is %d bytes", h.config.Upload.MaxFileSize))
		return
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	uniqueFileName := fmt.Sprintf("%s_%d_%s%s", baseName, timestamp, generateMessageID()[:8], ext)

	// Save file
	filePath := filepath.Join(h.config.Upload.UploadDir, uniqueFileName)
	if err := os.WriteFile(filePath, decodedData, 0644); err != nil {
		h.logger.WithError(err).Error("Failed to save file")
		h.sendError(client, "Failed to save file")
		return
	}

	// Create file URL
	fileURL := fmt.Sprintf("%s/%s", h.config.Upload.BaseURL, uniqueFileName)

	// Create message with file metadata
	message := &models.Message{
		ID:      generateMessageID(),
		Type:    models.FileMessage,
		Content: fmt.Sprintf("File: %s", fileName),
		Sender:  sender,
		Room:    roomID,
		Metadata: map[string]interface{}{
			"fileName": fileName,
			"fileURL":  fileURL,
			"fileType": fileType,
			"fileSize": len(decodedData),
		},
		Timestamp: time.Now(),
	}

	// Store message in Redis
	ctx := context.Background()
	if err := h.redisService.StoreMessage(ctx, message); err != nil {
		h.logger.WithError(err).Error("Failed to store file message")
		h.sendError(client, "Failed to send file")
		return
	}

	// Broadcast message
	h.broadcastMessage(message)

	h.logger.WithFields(logrus.Fields{
		"message_id": message.ID,
		"sender":     sender,
		"room_id":    roomID,
		"file_name":  fileName,
		"file_size":  len(decodedData),
	}).Info("File uploaded and message sent")
}

// broadcastMessage broadcasts a message using v4+ protocol
func (h *SocketIOHandler) broadcastMessage(message *models.Message) {
	if message.Room != "" {
		// Broadcast to room
		h.server.To(socket.Room(message.Room)).Emit("message", message)
	} else if message.Receiver != "" {
		// Direct message - 发送给指定用户的所有设备
		h.broadcastToUserDevices(message.Receiver, "message", map[string]interface{}{
			"message": message,
		}, "")
	} else {
		// Broadcast to all
		h.server.Emit("message", message)
	}
}

// broadcastUserStatus broadcasts user status changes
func (h *SocketIOHandler) broadcastUserStatus(userName, status string) {
	h.server.Emit("user_status", map[string]interface{}{
		"userName": userName,
		"status":   status,
	})
}

// sendError sends an error message to a specific client
func (h *SocketIOHandler) sendError(client *socket.Socket, message string) {
	client.Emit("error", map[string]interface{}{
		"message": message,
	})
}

// subscribeToRedis subscribes to Redis channels for distributed messaging
func (h *SocketIOHandler) subscribeToRedis() {
	ctx := context.Background()
	h.redisService.SubscribeToMessages(ctx, h.broadcastMessage)
}

// GetServer returns the Socket.IO server instance
func (h *SocketIOHandler) GetServer() *socket.Server {
	return h.server
}

// ServeHTTP handles HTTP requests for Socket.IO using v4+ protocol
func (h *SocketIOHandler) ServeHTTP(c *gin.Context) {
	handler := h.server.ServeHandler(nil)
	handler.ServeHTTP(c.Writer, c.Request)
}

// HandleFileUpload handles direct file upload via HTTP API
func (h *SocketIOHandler) HandleFileUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Check file size
	if file.Size > int64(h.config.Upload.MaxFileSize) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
		return
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	ext := filepath.Ext(file.Filename)
	baseName := strings.TrimSuffix(file.Filename, ext)
	uniqueFileName := fmt.Sprintf("%s_%d_%s%s", baseName, timestamp, generateMessageID()[:8], ext)

	// Save file
	filePath := filepath.Join(h.config.Upload.UploadDir, uniqueFileName)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		h.logger.WithError(err).Error("Failed to save uploaded file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Create file URL
	fileURL := fmt.Sprintf("%s/%s", h.config.Upload.BaseURL, uniqueFileName)

	c.JSON(http.StatusOK, gin.H{
		"fileURL":  fileURL,
		"fileName": file.Filename,
		"fileSize": file.Size,
	})
}

// GetOnlineUsers returns information about online users and their devices
func (h *SocketIOHandler) GetOnlineUsers() map[string]interface{} {
	users := make(map[string]interface{})

	for userName, sessions := range h.userSessions {
		devices := make([]map[string]interface{}, 0)
		for _, sessionID := range sessions {
			if user, ok := h.sessions[sessionID]; ok {
				deviceInfo, _ := user.Metadata["deviceInfo"].(string)
				devices = append(devices, map[string]interface{}{
					"sessionId":  sessionID,
					"deviceInfo": deviceInfo,
					"lastSeen":   user.LastSeen,
				})
			}
		}

		users[userName] = map[string]interface{}{
			"deviceCount": len(sessions),
			"devices":     devices,
			"status":      "online",
		}
	}

	return users
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), time.Now().Unix())
}
