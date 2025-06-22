package models

import (
	"encoding/json"
	"sync"
	"time"
)

// Quiz represents a quiz session
type Quiz struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Questions   []Question        `json:"questions"`
	Participants map[string]*User `json:"participants"`
	Status      QuizStatus        `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	EndedAt     *time.Time        `json:"ended_at,omitempty"`
	mu          sync.RWMutex      `json:"-"`
}

// QuizStatus represents the current status of a quiz
type QuizStatus string

const (
	QuizStatusWaiting QuizStatus = "waiting"
	QuizStatusActive  QuizStatus = "active"
	QuizStatusEnded   QuizStatus = "ended"
)

// Question represents a quiz question
type Question struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Options  []string `json:"options"`
	Correct  int      `json:"correct"`
	Points   int      `json:"points"`
	Category string   `json:"category"`
}

// User represents a participant in a quiz
type User struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Score    int       `json:"score"`
	Answers  []Answer  `json:"answers"`
	JoinedAt time.Time `json:"joined_at"`
	mu       sync.RWMutex `json:"-"`
}

// Answer represents a user's answer to a question
type Answer struct {
	QuestionID string    `json:"question_id"`
	Answer     int       `json:"answer"`
	Correct    bool      `json:"correct"`
	Points     int       `json:"points"`
	AnsweredAt time.Time `json:"answered_at"`
}

// LeaderboardEntry represents an entry in the leaderboard
type LeaderboardEntry struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Score    int    `json:"score"`
	Position int    `json:"position"`
}

// WebSocketMessage represents a message sent via WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// JoinQuizRequest represents a request to join a quiz
type JoinQuizRequest struct {
	QuizID string `json:"quiz_id"`
	Name   string `json:"name"`
}

// SubmitAnswerRequest represents a request to submit an answer
type SubmitAnswerRequest struct {
	QuizID     string `json:"quiz_id"`
	QuestionID string `json:"question_id"`
	Answer     int    `json:"answer"`
}

// QuizUpdate represents an update to the quiz state
type QuizUpdate struct {
	Type      string              `json:"type"`
	QuizID    string              `json:"quiz_id"`
	Leaderboard []LeaderboardEntry `json:"leaderboard,omitempty"`
	Question   *Question          `json:"question,omitempty"`
	UserScore  *UserScore         `json:"user_score,omitempty"`
}

// UserScore represents a user's score update
type UserScore struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Score  int    `json:"score"`
}

// Redis Keys
const (
	QuizKeyPrefix        = "quiz:"
	UserKeyPrefix        = "user:"
	LeaderboardKeyPrefix = "leaderboard:"
	ActiveQuizzesKey     = "active_quizzes"
)

// Methods for Quiz
func (q *Quiz) AddParticipant(user *User) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.Participants[user.ID] = user
}

func (q *Quiz) RemoveParticipant(userID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.Participants, userID)
}

func (q *Quiz) GetParticipants() map[string]*User {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.Participants
}

func (q *Quiz) GetLeaderboard() []LeaderboardEntry {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	entries := make([]LeaderboardEntry, 0, len(q.Participants))
	for _, user := range q.Participants {
		entries = append(entries, LeaderboardEntry{
			UserID: user.ID,
			Name:   user.Name,
			Score:  user.Score,
		})
	}
	
	// Sort by score (descending)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Score < entries[j].Score {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Add positions
	for i := range entries {
		entries[i].Position = i + 1
	}
	
	return entries
}

// Methods for User
func (u *User) AddAnswer(answer Answer) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Answers = append(u.Answers, answer)
	if answer.Correct {
		u.Score += answer.Points
	}
}

func (u *User) GetScore() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.Score
}

func (u *User) HasAnswered(questionID string) bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	for _, answer := range u.Answers {
		if answer.QuestionID == questionID {
			return true
		}
	}
	return false
}

// Redis serialization methods
func (q *Quiz) ToJSON() ([]byte, error) {
	return json.Marshal(q)
}

func (q *Quiz) FromJSON(data []byte) error {
	return json.Unmarshal(data, q)
}

func (u *User) ToJSON() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) FromJSON(data []byte) error {
	return json.Unmarshal(data, u)
} 