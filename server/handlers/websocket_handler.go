package handlers

import (
  "btaskee-quiz/models"
  "btaskee-quiz/services"
  "encoding/json"
  "log"
  "net/http"
  "sync"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/google/uuid"
  "github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
  quizService *services.QuizService
  upgrader    websocket.Upgrader
  mu          sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(quizService *services.QuizService) *WebSocketHandler {
  return &WebSocketHandler{
    quizService: quizService,
    upgrader: websocket.Upgrader{
      CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins for demo
      },
    },
  }
}

// HandleWebSocket handles WebSocket connections with Gin context
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
  // Get the underlying http.ResponseWriter and *http.Request from Gin
  w := c.Writer
  r := c.Request
  
  conn, err := h.upgrader.Upgrade(w, r, nil)
  if err != nil {
    log.Printf("WebSocket upgrade failed: %v", err)
    return
  }

  client := &services.Client{
    ID:   uuid.New().String()[:8],
    Send: make(chan []byte, 256),
    Hub:  h.quizService,
  }

  // Register client
  h.quizService.RegisterClient(client)

  // Start goroutines for reading and writing
  go h.readPump(client, conn)
  go h.writePump(client, conn)
}

// readPump pumps messages from the WebSocket connection to the hub
func (h *WebSocketHandler) readPump(client *services.Client, conn *websocket.Conn) {
  defer func() {
    h.quizService.UnregisterClient(client)
    conn.Close()
  }()

  conn.SetReadLimit(512) // Max message size
  conn.SetReadDeadline(time.Now().Add(60 * time.Second))
  conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    return nil
  })

  for {
    _, message, err := conn.ReadMessage()
    if err != nil {
      if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
        log.Printf("WebSocket read error: %v", err)
      }
      break
    }

    h.handleMessage(client, message)
  }
}

// writePump pumps messages from the hub to the WebSocket connection
func (h *WebSocketHandler) writePump(client *services.Client, conn *websocket.Conn) {
  ticker := time.NewTicker(54 * time.Second)
  defer func() {
    ticker.Stop()
    conn.Close()
  }()

  for {
    select {
    case message, ok := <-client.Send:
      conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
      if !ok {
        conn.WriteMessage(websocket.CloseMessage, []byte{})
        return
      }

      w, err := conn.NextWriter(websocket.TextMessage)
      if err != nil {
        return
      }
      w.Write(message)

      // Add queued messages to the current WebSocket message
      n := len(client.Send)
      for i := 0; i < n; i++ {
        w.Write([]byte{'\n'})
        w.Write(<-client.Send)
      }

      if err := w.Close(); err != nil {
        return
      }
    case <-ticker.C:
      conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
      if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
        return
      }
    }
  }
}

// handleMessage processes incoming WebSocket messages
func (h *WebSocketHandler) handleMessage(client *services.Client, message []byte) {
  var wsMessage models.WebSocketMessage
  err := json.Unmarshal(message, &wsMessage)
  if err != nil {
    log.Printf("Error unmarshaling message: %v", err)
    h.sendError(client, "Invalid message format")
    return
  }

  switch wsMessage.Type {
  case "join_quiz":
    h.handleJoinQuiz(client, wsMessage.Payload)
  case "submit_answer":
    h.handleSubmitAnswer(client, wsMessage.Payload)
  case "start_quiz":
    h.handleStartQuiz(client, wsMessage.Payload)
  case "end_quiz":
    h.handleEndQuiz(client, wsMessage.Payload)
  default:
    h.sendError(client, "Unknown message type: "+wsMessage.Type)
  }
}

// handleJoinQuiz handles join quiz requests
func (h *WebSocketHandler) handleJoinQuiz(client *services.Client, payload interface{}) {
  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    h.sendError(client, "Invalid payload")
    return
  }

  var joinRequest models.JoinQuizRequest
  err = json.Unmarshal(payloadBytes, &joinRequest)
  if err != nil {
    h.sendError(client, "Invalid join request")
    return
  }

  if joinRequest.QuizID == "" || joinRequest.Name == "" {
    h.sendError(client, "Quiz ID and name are required")
    return
  }

  // Join the quiz
  user, err := h.quizService.JoinQuiz(joinRequest.QuizID, joinRequest.Name)
  if err != nil {
    h.sendError(client, "Failed to join quiz: "+err.Error())
    return
  }

  // Update client info
  client.QuizID = joinRequest.QuizID
  client.UserID = user.ID

  // Send success response
  h.sendMessage(client, models.WebSocketMessage{
    Type: "join_success",
    Payload: map[string]interface{}{
      "user_id": user.ID,
      "name":    user.Name,
      "quiz_id": joinRequest.QuizID,
    },
  })

  // Send current quiz state
  quiz, err := h.quizService.GetQuiz(joinRequest.QuizID)
  if err == nil {
    h.sendMessage(client, models.WebSocketMessage{
      Type: "quiz_state",
      Payload: map[string]interface{}{
        "quiz":        quiz,
        "leaderboard": quiz.GetLeaderboard(),
      },
    })
  }

  log.Printf("👤 User %s joined quiz %s via WebSocket", user.Name, joinRequest.QuizID)
}

// handleSubmitAnswer handles answer submission
func (h *WebSocketHandler) handleSubmitAnswer(client *services.Client, payload interface{}) {
  if client.QuizID == "" || client.UserID == "" {
    h.sendError(client, "Must join a quiz first")
    return
  }

  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    h.sendError(client, "Invalid payload")
    return
  }

  var submitRequest models.SubmitAnswerRequest
  err = json.Unmarshal(payloadBytes, &submitRequest)
  if err != nil {
    h.sendError(client, "Invalid submit request")
    return
  }

  // Submit the answer
  err = h.quizService.SubmitAnswer(client.QuizID, client.UserID, submitRequest.QuestionID, submitRequest.Answer)
  if err != nil {
    h.sendError(client, "Failed to submit answer: "+err.Error())
    return
  }

  // Send success response
  h.sendMessage(client, models.WebSocketMessage{
    Type: "answer_submitted",
    Payload: map[string]interface{}{
      "question_id": submitRequest.QuestionID,
      "answer":      submitRequest.Answer,
    },
  })

  log.Printf("✅ Answer submitted for user %s, question %s", client.UserID, submitRequest.QuestionID)
}

// handleStartQuiz handles quiz start requests
func (h *WebSocketHandler) handleStartQuiz(client *services.Client, payload interface{}) {
  if client.QuizID == "" {
    h.sendError(client, "Must join a quiz first")
    return
  }

  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    h.sendError(client, "Invalid payload")
    return
  }

  var startRequest struct {
    QuizID string `json:"quiz_id"`
  }
  err = json.Unmarshal(payloadBytes, &startRequest)
  if err != nil {
    h.sendError(client, "Invalid start request")
    return
  }

  // Start the quiz
  err = h.quizService.StartQuiz(startRequest.QuizID)
  if err != nil {
    h.sendError(client, "Failed to start quiz: "+err.Error())
    return
  }

  log.Printf("🚀 Quiz %s started via WebSocket", startRequest.QuizID)
}

// handleEndQuiz handles quiz end requests
func (h *WebSocketHandler) handleEndQuiz(client *services.Client, payload interface{}) {
  if client.QuizID == "" {
    h.sendError(client, "Must join a quiz first")
    return
  }

  payloadBytes, err := json.Marshal(payload)
  if err != nil {
    h.sendError(client, "Invalid payload")
    return
  }

  var endRequest struct {
    QuizID string `json:"quiz_id"`
  }
  err = json.Unmarshal(payloadBytes, &endRequest)
  if err != nil {
    h.sendError(client, "Invalid end request")
    return
  }

  // End the quiz
  err = h.quizService.EndQuiz(endRequest.QuizID)
  if err != nil {
    h.sendError(client, "Failed to end quiz: "+err.Error())
    return
  }

  log.Printf("🏁 Quiz %s ended via WebSocket", endRequest.QuizID)
}

// sendMessage sends a message to a specific client
func (h *WebSocketHandler) sendMessage(client *services.Client, message models.WebSocketMessage) {
  data, err := json.Marshal(message)
  if err != nil {
    log.Printf("Error marshaling message: %v", err)
    return
  }

  select {
  case client.Send <- data:
  default:
    h.quizService.UnregisterClient(client)
  }
}

// sendError sends an error message to a client
func (h *WebSocketHandler) sendError(client *services.Client, errorMessage string) {
  h.sendMessage(client, models.WebSocketMessage{
    Type: "error",
    Payload: map[string]interface{}{
      "message": errorMessage,
    },
  })
}
