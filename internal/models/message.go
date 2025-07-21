package models

import (
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	TextMessage   MessageType = "text"
	FileMessage   MessageType = "file"
	ImageMessage  MessageType = "image"
	SystemMessage MessageType = "system"
)

// Message represents a chat message
type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	Sender    string      `json:"sender"`
	Receiver  string      `json:"receiver,omitempty"`
	Room      string      `json:"room,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// FileMetadata represents file-specific metadata
type FileMetadata struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileType string `json:"fileType"`
	FileURL  string `json:"fileURL"`
}

// User represents a connected user
type User struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Avatar   string                 `json:"avatar,omitempty"`
	Status   string                 `json:"status"`
	LastSeen time.Time              `json:"lastSeen"`
	Metadata map[string]interface{} `json:"metadata,omitempty"` // 用于存储设备信息等扩展数据
}

// Room represents a chat room
type Room struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	Members     []string  `json:"members"`
}

// Event represents different types of events
type Event string

const (
	EventJoin       Event = "join"
	EventLeave      Event = "leave"
	EventMessage    Event = "message"
	EventTyping     Event = "typing"
	EventStopTyping Event = "stop_typing"
	EventUserStatus Event = "user_status"
	EventRoomJoined Event = "room_joined"
	EventRoomLeft   Event = "room_left"
	EventError      Event = "error"
)

// SocketEvent represents a socket.io event
type SocketEvent struct {
	Event     Event       `json:"event"`
	Data      interface{} `json:"data"`
	Room      string      `json:"room,omitempty"`
	UserID    string      `json:"userId,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}
