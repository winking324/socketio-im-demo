package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"im-demo/internal/config"
	"im-demo/internal/models"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisService handles Redis operations
type RedisService struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewRedisService creates a new Redis service
func NewRedisService(cfg *config.Config, logger *logrus.Logger) (*RedisService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis successfully")

	return &RedisService{
		client: client,
		logger: logger,
	}, nil
}

// PublishMessage publishes a message to Redis
func (r *RedisService) PublishMessage(ctx context.Context, channel string, message *models.Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := r.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"channel":    channel,
		"message_id": message.ID,
		"sender":     message.Sender,
	}).Debug("Message published to Redis")

	return nil
}

// SubscribeToChannel subscribes to a Redis channel
func (r *RedisService) SubscribeToChannel(ctx context.Context, channel string, callback func(*models.Message)) error {
	pubsub := r.client.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Wait for subscription to be confirmed
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	r.logger.WithField("channel", channel).Info("Subscribed to Redis channel")

	// Start listening for messages
	ch := pubsub.Channel()
	for msg := range ch {
		var message models.Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal message")
			continue
		}

		callback(&message)
	}

	return nil
}

// StoreMessage stores a message in Redis with expiration
func (r *RedisService) StoreMessage(ctx context.Context, message *models.Message) error {
	key := fmt.Sprintf("message:%s", message.ID)
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Store message with 24 hour expiration
	if err := r.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	return nil
}

// GetMessage retrieves a message from Redis
func (r *RedisService) GetMessage(ctx context.Context, messageID string) (*models.Message, error) {
	key := fmt.Sprintf("message:%s", messageID)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var message models.Message
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &message, nil
}

// StoreUserSession stores user session information
func (r *RedisService) StoreUserSession(ctx context.Context, userID, sessionID string) error {
	key := fmt.Sprintf("user_session:%s", userID)
	if err := r.client.Set(ctx, key, sessionID, 12*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store user session: %w", err)
	}
	return nil
}

// GetUserSession retrieves user session information
func (r *RedisService) GetUserSession(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("user_session:%s", userID)
	sessionID, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("user session not found")
		}
		return "", fmt.Errorf("failed to get user session: %w", err)
	}
	return sessionID, nil
}

// DeleteUserSession deletes user session information
func (r *RedisService) DeleteUserSession(ctx context.Context, userID string) error {
	key := fmt.Sprintf("user_session:%s", userID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}
	return nil
}

// StoreRoomMembers stores room members
func (r *RedisService) StoreRoomMembers(ctx context.Context, roomID string, members []string) error {
	key := fmt.Sprintf("room_members:%s", roomID)
	if err := r.client.SAdd(ctx, key, members).Err(); err != nil {
		return fmt.Errorf("failed to store room members: %w", err)
	}
	return nil
}

// GetRoomMembers retrieves room members
func (r *RedisService) GetRoomMembers(ctx context.Context, roomID string) ([]string, error) {
	key := fmt.Sprintf("room_members:%s", roomID)
	members, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	return members, nil
}

// AddUserToRoom adds a user to a room
func (r *RedisService) AddUserToRoom(ctx context.Context, roomID, userID string) error {
	key := fmt.Sprintf("room_members:%s", roomID)
	if err := r.client.SAdd(ctx, key, userID).Err(); err != nil {
		return fmt.Errorf("failed to add user to room: %w", err)
	}
	return nil
}

// RemoveUserFromRoom removes a user from a room
func (r *RedisService) RemoveUserFromRoom(ctx context.Context, roomID, userID string) error {
	key := fmt.Sprintf("room_members:%s", roomID)
	if err := r.client.SRem(ctx, key, userID).Err(); err != nil {
		return fmt.Errorf("failed to remove user from room: %w", err)
	}
	return nil
}

// SubscribeToMessages subscribes to all message channels
func (r *RedisService) SubscribeToMessages(ctx context.Context, callback func(*models.Message)) {
	go func() {
		if err := r.SubscribeToChannel(ctx, "messages", callback); err != nil {
			r.logger.WithError(err).Error("Failed to subscribe to messages channel")
		}
	}()
}

// Close closes the Redis connection
func (r *RedisService) Close() error {
	return r.client.Close()
}
