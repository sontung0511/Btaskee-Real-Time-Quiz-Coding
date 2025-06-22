package main

import (
  "btaskee-quiz/handlers"
  "btaskee-quiz/services"
  "log"
  "net/http"
  "os"

  "github.com/gin-contrib/cors"
  "github.com/gin-gonic/gin"
)

func main() {
  log.Printf("Starting Btaskee Real-Time Quiz with Redis...")

  // Initialize Redis service
  redisService := services.NewRedisService()

  // Initialize quiz service
  quizService := services.NewQuizService(redisService)

  // Initialize handlers
  httpHandler := handlers.NewHTTPHandler(quizService)
  wsHandler := handlers.NewWebSocketHandler(quizService)

  // Setup Gin router
  router := gin.Default()

  // CORS configuration
  config := cors.DefaultConfig()
  config.AllowAllOrigins = true
  config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
  config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-ID"}
  router.Use(cors.New(config))

  // Web interface route
  router.GET("/", func(c *gin.Context) {
    c.HTML(http.StatusOK, "index.html", gin.H{
      "title": "Btaskee Real-Time Quiz",
    })
  })

  // API routes
  api := router.Group("/api/v1")
  {
    // Quiz management
    // POST /api/v1/quizzes - Create a new quiz
    api.POST("/quizzes", httpHandler.CreateQuiz)

    // GET /api/v1/quizzes - Get all active quizzes
    api.GET("/quizzes", httpHandler.GetActiveQuizzes)

    // GET /api/v1/quizzes/:id - Get quiz details
    api.GET("/quizzes/:id", httpHandler.GetQuiz)

    // DELETE /api/v1/quizzes/:id - Delete a quiz
    api.DELETE("/quizzes/:id", httpHandler.DeleteQuiz)

    // Quiz participation
    // POST /api/v1/quizzes/join - Join a quiz
    api.POST("/quizzes/join", httpHandler.JoinQuiz)

    // POST /api/v1/quizzes/answer - Submit an answer
    api.POST("/quizzes/answer", httpHandler.SubmitAnswer)

    // GET /api/v1/quizzes/:id/leaderboard - Get leaderboard
    api.GET("/quizzes/:id/leaderboard", httpHandler.GetLeaderboard)

    // Quiz control
    // POST /api/v1/quizzes/:id/start - Start a quiz
    api.POST("/quizzes/:id/start", httpHandler.StartQuiz)

    // POST /api/v1/quizzes/:id/end - End a quiz
    api.POST("/quizzes/:id/end", httpHandler.EndQuiz)

    // Health check
    // GET /api/v1/health - Health check endpoint
    api.GET("/health", httpHandler.HealthCheck)
  }

  // WebSocket endpoint
  // GET /ws - WebSocket connection for real-time updates
  router.GET("/ws", wsHandler.HandleWebSocket)

  // Get port from environment or use default
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }

  log.Printf("Starting server on port :%s", port)
  log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
  log.Printf("API endpoint: http://localhost:%s/api/v1", port)
  log.Printf("Web interface: http://localhost:%s", port)

  // Start server
  if err := router.Run(":" + port); err != nil {
    log.Fatal("Failed to start server:", err)
  }
}
