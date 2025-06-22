package handlers

import (
  "btaskee-quiz/models"
  "btaskee-quiz/services"
  "net/http"

  "github.com/gin-gonic/gin"
)

// HTTPHandler handles HTTP API requests
type HTTPHandler struct {
  quizService *services.QuizService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(quizService *services.QuizService) *HTTPHandler {
  return &HTTPHandler{
    quizService: quizService,
  }
}

// CreateQuiz creates a new quiz
// APi /api/v1/quizzes [POST]
func (h *HTTPHandler) CreateQuiz(c *gin.Context) {
  var request struct {
    Title string `json:"title" binding:"required"`
  }

  if err := c.ShouldBindJSON(&request); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Title is required",
    })
    return
  }

  quiz, err := h.quizService.CreateQuiz(request.Title)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to create quiz: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusCreated, gin.H{
    "message": "Quiz created successfully",
    "quiz":    quiz,
  })
}

// GetQuiz retrieves a quiz by ID
// APi /api/v1/quizzes/:id
func (h *HTTPHandler) GetQuiz(c *gin.Context) {
  quizID := c.Param("id")
  if quizID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID is required",
    })
    return
  }

  quiz, err := h.quizService.GetQuiz(quizID)
  if err != nil {
    c.JSON(http.StatusNotFound, gin.H{
      "error": "Quiz not found: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "quiz": quiz,
  })
}

// JoinQuiz allows a user to join a quiz
func (h *HTTPHandler) JoinQuiz(c *gin.Context) {
  var request models.JoinQuizRequest

  if err := c.ShouldBindJSON(&request); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID and name are required",
    })
    return
  }

  user, err := h.quizService.JoinQuiz(request.QuizID, request.Name)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Failed to join quiz: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "message": "Successfully joined quiz",
    "user":    user,
  })
}

// SubmitAnswer submits an answer to a question
func (h *HTTPHandler) SubmitAnswer(c *gin.Context) {
  var request models.SubmitAnswerRequest

  if err := c.ShouldBindJSON(&request); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID, question ID, and answer are required",
    })
    return
  }

  // Get user ID from query parameter or header
  userID := c.Query("user_id")
  if userID == "" {
    userID = c.GetHeader("X-User-ID")
  }

  if userID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "User ID is required",
    })
    return
  }

  err := h.quizService.SubmitAnswer(request.QuizID, userID, request.QuestionID, request.Answer)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Failed to submit answer: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "message": "Answer submitted successfully",
  })
}

// GetLeaderboard retrieves the leaderboard for a quiz
func (h *HTTPHandler) GetLeaderboard(c *gin.Context) {
  quizID := c.Param("id")
  if quizID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID is required",
    })
    return
  }

  leaderboard, err := h.quizService.GetLeaderboard(quizID)
  if err != nil {
    c.JSON(http.StatusNotFound, gin.H{
      "error": "Failed to get leaderboard: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "leaderboard": leaderboard,
  })
}

// StartQuiz starts a quiz session
func (h *HTTPHandler) StartQuiz(c *gin.Context) {
  quizID := c.Param("id")
  if quizID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID is required",
    })
    return
  }

  err := h.quizService.StartQuiz(quizID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Failed to start quiz: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "message": "Quiz started successfully",
  })
}

// EndQuiz ends a quiz session
func (h *HTTPHandler) EndQuiz(c *gin.Context) {
  quizID := c.Param("id")
  if quizID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID is required",
    })
    return
  }

  err := h.quizService.EndQuiz(quizID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Failed to end quiz: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "message": "Quiz ended successfully",
  })
}

// GetUser retrieves user information
func (h *HTTPHandler) GetUser(c *gin.Context) {
  userID := c.Param("id")
  if userID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "User ID is required",
    })
    return
  }

  user, err := h.quizService.RedisService.GetUser(userID)
  if err != nil {
    c.JSON(http.StatusNotFound, gin.H{
      "error": "User not found: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "user": user,
  })
}

// HealthCheck provides health check endpoint
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
  redisStatus := "connected"
  if !h.quizService.RedisService.IsAvailable() {
    redisStatus = "disconnected"
  }

  c.JSON(http.StatusOK, gin.H{
    "status":  "healthy",
    "redis":   redisStatus,
    "quizzes": len(h.quizService.Quizzes),
    "clients": len(h.quizService.Clients),
  })
}

// GetActiveQuizzes returns all active quizzes
func (h *HTTPHandler) GetActiveQuizzes(c *gin.Context) {
  activeQuizzes, err := h.quizService.RedisService.GetActiveQuizzes()
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to get active quizzes: " + err.Error(),
    })
    return
  }

  quizzes := make([]*models.Quiz, 0)
  for _, quizID := range activeQuizzes {
    quiz, err := h.quizService.GetQuiz(quizID)
    if err != nil {
      continue // Skip if quiz can't be loaded
    }
    quizzes = append(quizzes, quiz)
  }

  c.JSON(http.StatusOK, gin.H{
    "quizzes": quizzes,
    "count":   len(quizzes),
  })
}

// DeleteQuiz deletes a quiz
func (h *HTTPHandler) DeleteQuiz(c *gin.Context) {
  quizID := c.Param("id")
  if quizID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID is required",
    })
    return
  }

  // Remove from memory
  h.quizService.Mu.Lock()
  delete(h.quizService.Quizzes, quizID)
  h.quizService.Mu.Unlock()

  // Remove from Redis
  err := h.quizService.RedisService.DeleteQuiz(quizID)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to delete quiz: " + err.Error(),
    })
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "message": "Quiz deleted successfully",
  })
}

// GetQuizStats returns statistics for a quiz
func (h *HTTPHandler) GetQuizStats(c *gin.Context) {
  quizID := c.Param("id")
  if quizID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
      "error": "Quiz ID is required",
    })
    return
  }

  quiz, err := h.quizService.GetQuiz(quizID)
  if err != nil {
    c.JSON(http.StatusNotFound, gin.H{
      "error": "Quiz not found: " + err.Error(),
    })
    return
  }

  // Calculate statistics
  totalParticipants := len(quiz.Participants)
  totalQuestions := len(quiz.Questions)

  var totalAnswers int
  var correctAnswers int
  var totalScore int

  for _, user := range quiz.Participants {
    totalAnswers += len(user.Answers)
    for _, answer := range user.Answers {
      if answer.Correct {
        correctAnswers++
      }
    }
    totalScore += user.Score
  }

  accuracy := 0.0
  if totalAnswers > 0 {
    accuracy = float64(correctAnswers) / float64(totalAnswers) * 100
  }

  avgScore := 0.0
  if totalParticipants > 0 {
    avgScore = float64(totalScore) / float64(totalParticipants)
  }

  c.JSON(http.StatusOK, gin.H{
    "quiz_id":            quizID,
    "title":              quiz.Title,
    "status":             quiz.Status,
    "total_participants": totalParticipants,
    "total_questions":    totalQuestions,
    "total_answers":      totalAnswers,
    "correct_answers":    correctAnswers,
    "accuracy_percent":   accuracy,
    "total_score":        totalScore,
    "average_score":      avgScore,
    "created_at":         quiz.CreatedAt,
    "started_at":         quiz.StartedAt,
    "ended_at":           quiz.EndedAt,
  })
}
