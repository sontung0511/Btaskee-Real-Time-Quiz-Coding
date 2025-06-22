package services

import (
  "btaskee-quiz/models"
  "context"
  "encoding/json"
  "fmt"
  "log"
  "sync"
  "time"

  "github.com/google/uuid"
)

// QuizService manages quiz sessions
type QuizService struct {
  Quizzes      map[string]*models.Quiz
  Clients      map[*Client]bool
  RedisService *RedisService
  Mu           sync.RWMutex // Keep for Clients map only
}

// Client represents a WebSocket client
type Client struct {
  ID     string
  QuizID string
  UserID string
  Send   chan []byte
  Hub    *QuizService
}

// NewQuizService creates a new quiz service
func NewQuizService(redisService *RedisService) *QuizService {
  qs := &QuizService{
    Quizzes:      make(map[string]*models.Quiz),
    Clients:      make(map[*Client]bool),
    RedisService: redisService,
  }

  // Load existing quizzes from Redis
  qs.loadQuizzesFromRedis()

  // Start Redis subscription for cross-instance communication
  go qs.startRedisSubscription()

  return qs
}

// CreateQuiz creates a new quiz session
func (qs *QuizService) CreateQuiz(title string) (*models.Quiz, error) {
  quizID := generateQuizID()
  quiz := &models.Quiz{
    ID:           quizID,
    Title:        title,
    Questions:    getSampleQuestions(),
    Participants: make(map[string]*models.User),
    Status:       models.QuizStatusWaiting,
    CreatedAt:    time.Now(),
  }

  // Save to Redis first
  err := qs.RedisService.SaveQuiz(quiz)
  if err != nil {
    return nil, fmt.Errorf("failed to save quiz to Redis: %v", err)
  }

  // Then add to memory
  qs.Quizzes[quizID] = quiz

  log.Printf("🎯 Created quiz: %s (%s)", title, quizID)
  return quiz, nil
}

// GetQuiz retrieves a quiz by ID
func (qs *QuizService) GetQuiz(quizID string) (*models.Quiz, error) {
  // Try memory first
  if quiz, exists := qs.Quizzes[quizID]; exists {
    return quiz, nil
  }

  // Try to load from Redis
  quiz, err := qs.RedisService.GetQuiz(quizID)
  if err != nil {
    return nil, fmt.Errorf("quiz not found: %s", quizID)
  }

  // Add to memory
  qs.Quizzes[quizID] = quiz
  return quiz, nil
}

// JoinQuiz allows a user to join a quiz session
func (qs *QuizService) JoinQuiz(quizID, userName string) (*models.User, error) {
  quiz, err := qs.GetQuiz(quizID)
  if err != nil {
    return nil, err
  }

  userID := generateUserID()
  user := &models.User{
    ID:       userID,
    Name:     userName,
    Score:    0,
    Answers:  []models.Answer{},
    JoinedAt: time.Now(),
  }

  quiz.AddParticipant(user)

  // Save to Redis
  err = qs.RedisService.SaveQuiz(quiz)
  if err != nil {
    log.Printf("Warning: failed to save quiz to Redis: %v", err)
  }

  err = qs.RedisService.SaveUser(user)
  if err != nil {
    log.Printf("Warning: failed to save user to Redis: %v", err)
  }

  // Broadcast join event
  qs.broadcastToQuiz(quizID, models.WebSocketMessage{
    Type: "user_joined",
    Payload: map[string]interface{}{
      "user_id": userID,
      "name":    userName,
    },
  })

  // Broadcast updated leaderboard
  qs.broadcastLeaderboard(quizID)

  log.Printf("👤 User %s joined quiz %s", userName, quizID)
  return user, nil
}

// SubmitAnswer processes a user's answer
func (qs *QuizService) SubmitAnswer(quizID, userID, questionID string, answer int) error {
  quiz, err := qs.GetQuiz(quizID)
  if err != nil {
    return err
  }

  user, exists := quiz.Participants[userID]
  if !exists {
    return fmt.Errorf("user not found: %s", userID)
  }

  // Check if user already answered this question
  if user.HasAnswered(questionID) {
    return fmt.Errorf("user already answered this question")
  }

  // Find the question
  var question *models.Question
  for _, q := range quiz.Questions {
    if q.ID == questionID {
      question = &q
      break
    }
  }

  if question == nil {
    return fmt.Errorf("question not found: %s", questionID)
  }

  // Check if answer is correct
  isCorrect := answer == question.Correct
  points := 0
  if isCorrect {
    points = question.Points
  }

  // Create answer record
  answerRecord := models.Answer{
    QuestionID: questionID,
    Answer:     answer,
    Correct:    isCorrect,
    Points:     points,
    AnsweredAt: time.Now(),
  }

  // Add answer to user
  user.AddAnswer(answerRecord)

  // Save to Redis
  err = qs.RedisService.SaveQuiz(quiz)
  if err != nil {
    log.Printf("Warning: failed to save quiz to Redis: %v", err)
  }

  err = qs.RedisService.SaveUser(user)
  if err != nil {
    log.Printf("Warning: failed to save user to Redis: %v", err)
  }

  // Broadcast score update
  qs.broadcastToQuiz(quizID, models.WebSocketMessage{
    Type: "score_update",
    Payload: models.UserScore{
      UserID: userID,
      Name:   user.Name,
      Score:  user.GetScore(),
    },
  })

  // Broadcast updated leaderboard
  qs.broadcastLeaderboard(quizID)

  log.Printf("✅ User %s answered question %s (correct: %v, points: %d)",
    user.Name, questionID, isCorrect, points)
  return nil
}

// GetLeaderboard returns the current leaderboard for a quiz
func (qs *QuizService) GetLeaderboard(quizID string) ([]models.LeaderboardEntry, error) {
  quiz, err := qs.GetQuiz(quizID)
  if err != nil {
    return nil, err
  }

  leaderboard := quiz.GetLeaderboard()

  // Save to Redis (async to avoid blocking)
  go func() {
    err := qs.RedisService.SaveLeaderboard(quizID, leaderboard)
    if err != nil {
      log.Printf("Warning: failed to save leaderboard to Redis: %v", err)
    }
  }()

  return leaderboard, nil
}

// StartQuiz starts a quiz session
func (qs *QuizService) StartQuiz(quizID string) error {
  quiz, err := qs.GetQuiz(quizID)
  if err != nil {
    return err
  }

  if quiz.Status != models.QuizStatusWaiting {
    return fmt.Errorf("quiz is not in waiting status")
  }

  now := time.Now()
  quiz.Status = models.QuizStatusActive
  quiz.StartedAt = &now

  // Save to Redis
  err = qs.RedisService.SaveQuiz(quiz)
  if err != nil {
    log.Printf("Warning: failed to save quiz to Redis: %v", err)
  }

  // Broadcast quiz start
  qs.broadcastToQuiz(quizID, models.WebSocketMessage{
    Type: "quiz_started",
    Payload: map[string]interface{}{
      "quiz_id":    quizID,
      "started_at": now, 
    },
  })

  log.Printf("🚀 Quiz %s started", quizID)
  return nil
}

// EndQuiz ends a quiz session
func (qs *QuizService) EndQuiz(quizID string) error {
  quiz, err := qs.GetQuiz(quizID)
  if err != nil {
    return err
  }

  now := time.Now()
  quiz.Status = models.QuizStatusEnded
  quiz.EndedAt = &now

  // Save to Redis
  err = qs.RedisService.SaveQuiz(quiz)
  if err != nil {
    log.Printf("Warning: failed to save quiz to Redis: %v", err)
  }

  // Broadcast quiz end
  qs.broadcastToQuiz(quizID, models.WebSocketMessage{
    Type: "quiz_ended",
    Payload: map[string]interface{}{
      "quiz_id":  quizID,
      "ended_at": now,
    },
  })

  log.Printf("🏁 Quiz %s ended", quizID)
  return nil
}

// RegisterClient registers a WebSocket client
func (qs *QuizService) RegisterClient(client *Client) {
  qs.Mu.Lock()
  defer qs.Mu.Unlock()
  qs.Clients[client] = true
  log.Printf("🔌 Client %s registered for quiz %s", client.ID, client.QuizID)
}

// UnregisterClient unregisters a WebSocket client
func (qs *QuizService) UnregisterClient(client *Client) {
  qs.Mu.Lock()
  defer qs.Mu.Unlock()
  if _, ok := qs.Clients[client]; ok {
    delete(qs.Clients, client)
    close(client.Send)
    log.Printf("🔌 Client %s unregistered", client.ID)
  }
}

// broadcastToQuiz sends a message to all Clients in a quiz
func (qs *QuizService) broadcastToQuiz(quizID string, message models.WebSocketMessage) {
  data, err := json.Marshal(message)
  if err != nil {
    log.Printf("Error marshaling message: %v", err)
    return
  }

  // Collect clients to remove
  clientsToRemove := make([]*Client, 0)

  qs.Mu.RLock()
  for client := range qs.Clients {
    if client.QuizID == quizID {
      select {
      case client.Send <- data:
        // Message sent successfully
      default:
        // Channel is full or closed, mark for removal
        clientsToRemove = append(clientsToRemove, client)
      }
    }
  }
  qs.Mu.RUnlock()

  // Remove dead clients with write lock
  if len(clientsToRemove) > 0 {
    qs.Mu.Lock()
    for _, client := range clientsToRemove {
      if _, ok := qs.Clients[client]; ok {
        delete(qs.Clients, client)
        close(client.Send)
        log.Printf("🔌 Removed dead client %s", client.ID)
      }
    }
    qs.Mu.Unlock()
  }

  // Publish to Redis for cross-instance communication
  err = qs.RedisService.PublishMessage("quiz:"+quizID, message)
  if err != nil {
    log.Printf("Warning: failed to publish to Redis: %v", err)
  }
}

// broadcastLeaderboard broadcasts the current leaderboard
func (qs *QuizService) broadcastLeaderboard(quizID string) {
  leaderboard, err := qs.GetLeaderboard(quizID)
  if err != nil {
    log.Printf("Error getting leaderboard: %v", err)
    return
  }

  qs.broadcastToQuiz(quizID, models.WebSocketMessage{
    Type:    "leaderboard_update",
    Payload: leaderboard,
  })
}

// loadQuizzesFromRedis loads existing Quizzes from Redis
func (qs *QuizService) loadQuizzesFromRedis() {
  if !qs.RedisService.IsAvailable() {
    return
  }

  activeQuizzes, err := qs.RedisService.GetActiveQuizzes()
  if err != nil {
    log.Printf("Warning: failed to get active quizzes: %v", err)
    return
  }

  for _, quizID := range activeQuizzes {
    quiz, err := qs.RedisService.GetQuiz(quizID)
    if err != nil {
      log.Printf("Warning: failed to load quiz %s: %v", quizID, err)
      continue
    }

    qs.Quizzes[quizID] = quiz
    log.Printf("📂 Loaded quiz %s from Redis", quizID)
  }

  log.Printf("📂 Loaded %d quizzes from Redis", len(activeQuizzes))
}

// startRedisSubscription starts listening for Redis pub/sub messages
func (qs *QuizService) startRedisSubscription() {
  if !qs.RedisService.IsAvailable() {
    return
  }

  pubSub := qs.RedisService.SubscribeToChannel("quiz:*")
  defer pubSub.Close()

  ctx := context.Background()
  for {
    msg, err := pubSub.ReceiveMessage(ctx)
    if err != nil {
      log.Printf("Redis subscription error: %v", err)
      break
    }

    var message models.WebSocketMessage
    err = json.Unmarshal([]byte(msg.Payload), &message)
    if err != nil {
      log.Printf("Error unmarshaling Redis message: %v", err)
      continue
    }

    // Extract quiz ID from channel name
    quizID := msg.Channel[5:] // Remove "quiz:" prefix

    // Broadcast to local clients
    qs.broadcastToQuiz(quizID, message)
  }
}

// Helper functions
func generateQuizID() string {
  return uuid.New().String()[:8]
}

func generateUserID() string {
  return uuid.New().String()[:8]
}

// getSampleQuestions returns sample quiz questions
func getSampleQuestions() []models.Question {
  return []models.Question{
    {
      ID:       "q1",
      Text:     "What is the capital of Vietnam?",
      Options:  []string{"Hanoi", "Ho Chi Minh City", "Da Nang", "Hue"},
      Correct:  0,
      Points:   10,
      Category: "Geography",
    },
    {
      ID:       "q2",
      Text:     "Which programming language is this quiz written in?",
      Options:  []string{"Python", "JavaScript", "Go", "Java"},
      Correct:  2,
      Points:   15,
      Category: "Programming",
    },
    {
      ID:       "q3",
      Text:     "What is Redis primarily used for?",
      Options:  []string{"File storage", "In-memory data store", "Database backup", "Email service"},
      Correct:  1,
      Points:   20,
      Category: "Technology",
    },
    {
      ID:       "q4",
      Text:     "What does WebSocket provide?",
      Options:  []string{"File upload", "Real-time communication", "Database queries", "Email sending"},
      Correct:  1,
      Points:   15,
      Category: "Technology",
    },
    {
      ID:       "q5",
      Text:     "Which company owns Btaskee?",
      Options:  []string{"Grab", "GoJek", "Btaskee Pte Ltd", "Lazada"},
      Correct:  2,
      Points:   10,
      Category: "Business",
    },
  }
}
