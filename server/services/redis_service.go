package services

import (
	"btaskee-quiz/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisService handles all Redis operations
type RedisService struct {
	client *redis.Client
}

// NewRedisService creates a new Redis service
func NewRedisService() *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // no password set
		DB:       0,                // use default DB
	})

	// Test connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Redis connection failed: %v", err)
		log.Printf("Running in memory-only mode")
		return &RedisService{client: nil}
	}

	log.Printf("Connected to Redis successfully")
	return &RedisService{client: client}
}

// SaveQuiz saves a quiz to Redis
func (rs *RedisService) SaveQuiz(quiz *models.Quiz) error {
	if rs.client == nil {
		return nil // Skip if Redis is not available
	}

	ctx := context.Background()
	quizData, err := json.Marshal(quiz)
	if err != nil {
		return fmt.Errorf("failed to marshal quiz: %v", err)
	}

	key := models.QuizKeyPrefix + quiz.ID
	err = rs.client.Set(ctx, key, quizData, 24*time.Hour).Err() // Expire after 24 hours
	if err != nil {
		return fmt.Errorf("failed to save quiz to Redis: %v", err)
	}

	// Add to active quizzes set
	err = rs.client.SAdd(ctx, models.ActiveQuizzesKey, quiz.ID).Err()
	if err != nil {
		log.Printf("Warning: failed to add quiz to active set: %v", err)
	}

	log.Printf("💾 Saved quiz %s to Redis", quiz.ID)
	return nil
}

// GetQuiz retrieves a quiz from Redis
func (rs *RedisService) GetQuiz(quizID string) (*models.Quiz, error) {
	if rs.client == nil {
		return nil, fmt.Errorf("Redis not available")
	}

	ctx := context.Background()
	key := models.QuizKeyPrefix + quizID
	quizData, err := rs.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("quiz not found: %s", quizID)
		}
		return nil, fmt.Errorf("failed to get quiz from Redis: %v", err)
	}

	var quiz models.Quiz
	err = json.Unmarshal([]byte(quizData), &quiz)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal quiz: %v", err)
	}

	return &quiz, nil
}

// SaveUser saves a user to Redis
func (rs *RedisService) SaveUser(user *models.User) error {
	if rs.client == nil {
		return nil
	}

	ctx := context.Background()
	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %v", err)
	}

	key := models.UserKeyPrefix + user.ID
	err = rs.client.Set(ctx, key, userData, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save user to Redis: %v", err)
	}

	return nil
}

// GetUser retrieves a user from Redis
func (rs *RedisService) GetUser(userID string) (*models.User, error) {
	if rs.client == nil {
		return nil, fmt.Errorf("Redis not available")
	}

	ctx := context.Background()
	key := models.UserKeyPrefix + userID
	userData, err := rs.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found: %s", userID)
		}
		return nil, fmt.Errorf("failed to get user from Redis: %v", err)
	}

	var user models.User
	err = json.Unmarshal([]byte(userData), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %v", err)
	}

	return &user, nil
}

// SaveLeaderboard saves leaderboard to Redis
func (rs *RedisService) SaveLeaderboard(quizID string, leaderboard []models.LeaderboardEntry) error {
	if rs.client == nil {
		return nil
	}

	ctx := context.Background()
	leaderboardData, err := json.Marshal(leaderboard)
	if err != nil {
		return fmt.Errorf("failed to marshal leaderboard: %v", err)
	}

	key := models.LeaderboardKeyPrefix + quizID
	err = rs.client.Set(ctx, key, leaderboardData, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save leaderboard to Redis: %v", err)
	}

	return nil
}

// GetLeaderboard retrieves leaderboard from Redis
func (rs *RedisService) GetLeaderboard(quizID string) ([]models.LeaderboardEntry, error) {
	if rs.client == nil {
		return nil, fmt.Errorf("Redis not available")
	}

	ctx := context.Background()
	key := models.LeaderboardKeyPrefix + quizID
	leaderboardData, err := rs.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return []models.LeaderboardEntry{}, nil
		}
		return nil, fmt.Errorf("failed to get leaderboard from Redis: %v", err)
	}

	var leaderboard []models.LeaderboardEntry
	err = json.Unmarshal([]byte(leaderboardData), &leaderboard)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal leaderboard: %v", err)
	}

	return leaderboard, nil
}

// GetActiveQuizzes retrieves all active quiz IDs
func (rs *RedisService) GetActiveQuizzes() ([]string, error) {
	if rs.client == nil {
		return []string{}, nil
	}

	ctx := context.Background()
	quizIDs, err := rs.client.SMembers(ctx, models.ActiveQuizzesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active quizzes: %v", err)
	}

	return quizIDs, nil
}

// DeleteQuiz removes a quiz from Redis
func (rs *RedisService) DeleteQuiz(quizID string) error {
	if rs.client == nil {
		return nil
	}

	ctx := context.Background()
	
	// Remove quiz data
	quizKey := models.QuizKeyPrefix + quizID
	err := rs.client.Del(ctx, quizKey).Err()
	if err != nil {
		log.Printf("Warning: failed to delete quiz data: %v", err)
	}

	// Remove leaderboard
	leaderboardKey := models.LeaderboardKeyPrefix + quizID
	err = rs.client.Del(ctx, leaderboardKey).Err()
	if err != nil {
		log.Printf("Warning: failed to delete leaderboard: %v", err)
	}

	// Remove from active quizzes set
	err = rs.client.SRem(ctx, models.ActiveQuizzesKey, quizID).Err()
	if err != nil {
		log.Printf("Warning: failed to remove from active quizzes: %v", err)
	}

	log.Printf("🗑️  Deleted quiz %s from Redis", quizID)
	return nil
}

// PublishMessage publishes a message to Redis pub/sub
func (rs *RedisService) PublishMessage(channel string, message interface{}) error {
	if rs.client == nil {
		return nil
	}

	ctx := context.Background()
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	err = rs.client.Publish(ctx, channel, messageData).Err()
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}

// SubscribeToChannel subscribes to a Redis channel
func (rs *RedisService) SubscribeToChannel(channel string) *redis.PubSub {
	if rs.client == nil {
		return nil
	}

	return rs.client.Subscribe(context.Background(), channel)
}

// Close closes the Redis connection
func (rs *RedisService) Close() error {
	if rs.client == nil {
		return nil
	}

	return rs.client.Close()
}

// IsAvailable checks if Redis is available
func (rs *RedisService) IsAvailable() bool {
	return rs.client != nil
} 