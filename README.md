# 🎯 Btaskee Real-Time Quiz with Redis

A high-performance, scalable real-time quiz application built with Go, WebSocket, and Redis. Perfect for interactive learning, team building, and competitive assessments.

## ✨ Features

### 🚀 Real-Time Capabilities
- **Live Score Updates**: Instant score synchronization across all participants
- **Real-Time Leaderboard**: Dynamic leaderboard that updates as users answer questions
- **WebSocket Communication**: Low-latency bidirectional communication
- **Multi-User Support**: Multiple users can join and participate simultaneously

### 💾 Data Persistence & Scalability
- **Redis Integration**: Fast in-memory data storage with persistence
- **Cross-Instance Sync**: Multiple server instances can communicate via Redis Pub/Sub
- **Graceful Degradation**: Falls back to memory-only mode if Redis is unavailable
- **Horizontal Scaling**: Support for load balancing across multiple instances

### 🎮 Quiz Management
- **Dynamic Quiz Creation**: Create quizzes with custom titles
- **Sample Questions**: Pre-loaded questions covering various topics
- **Quiz States**: Waiting, Active, and Ended states
- **Answer Validation**: Prevents duplicate answers and validates responses

### 🏆 User Experience
- **Beautiful UI**: Modern, responsive web interface
- **Real-Time Notifications**: Live updates for all quiz events
- **Connection Status**: Visual indicators for WebSocket connection
- **Mobile Responsive**: Works seamlessly on all devices


## 🚀 Quick Start

### Prerequisites
- Go 1.21 or higher
- Redis server (optional, will run in memory-only mode if not available)

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd Btaskee-Real-Time-Quiz-Coding
```

2. **Install dependencies**
```bash
go mod tidy
```

3. **Start Redis (optional)**
```bash
# macOS
brew install redis
redis-server

# Ubuntu/Debian
sudo apt-get install redis-server
redis-server

# Windows
# Download Redis from https://redis.io/download
```

4. **Run the application**
```bash
go run main.go
```

5. **Access the application**
- Web Interface: http://localhost:8080
- API Documentation: http://localhost:8080/api/v1
- WebSocket Endpoint: ws://localhost:8080/ws

## 📡 API Endpoints

### Quiz Management
- `POST /api/v1/quizzes` - Create a new quiz
- `GET /api/v1/quizzes` - Get all active quizzes
- `GET /api/v1/quizzes/:id` - Get quiz details
- `DELETE /api/v1/quizzes/:id` - Delete a quiz

### Quiz Participation
- `POST /api/v1/quizzes/join` - Join a quiz
- `POST /api/v1/quizzes/answer` - Submit an answer
- `GET /api/v1/quizzes/:id/leaderboard` - Get leaderboard

### Quiz Control
- `POST /api/v1/quizzes/:id/start` - Start a quiz
- `POST /api/v1/quizzes/:id/end` - End a quiz

### Health & Monitoring
- `GET /api/v1/health` - Health check endpoint

## 🔌 WebSocket Messages

```json
// Join quiz
{
  "type": "join_quiz",
  "payload": {
    "quiz_id": "abc123",
    "name": "John Doe"
  }
}

// Submit answer
{
  "type": "submit_answer",
  "payload": {
    "quiz_id": "abc123",
    "question_id": "q1",
    "answer": 2
  }
}

// Start quiz
{
  "type": "start_quiz",
  "payload": {
    "quiz_id": "abc123"
  }
}
```


```
```bash
# Terminal 1
go run main.go

# Terminal 2
PORT=8081 go run main.go
```
1. Create quiz in instance 1
2. Join quiz in instance 2
3. Verify real-time sync between instances

## 🔧 Configuration

### Environment Variables
- `PORT`: Server port (default: 8080)
- `REDIS_ADDR`: Redis server address (default: localhost:6379)
- `REDIS_PASSWORD`: Redis password (default: empty)
- `REDIS_DB`: Redis database number (default: 0)

### Redis Configuration
The application automatically detects Redis availability:
- **Redis Available**: Full functionality with persistence and cross-instance sync
- **Redis Unavailable**: Memory-only mode with graceful degradation

## 🏗️ Project Structure

```
├── main.go                 # Application entry point
├── go.mod                  # Go module dependencies
├── models/
│   └── quiz.go            # Data models and structures
├── services/
│   ├── redis_service.go   # Redis operations and persistence
│   └── quiz_service.go    # Quiz business logic
├── handlers/
│   ├── http_handler.go    # HTTP API endpoints
│   └── websocket_handler.go # WebSocket communication
├── templates/
│   └── index.html         # Web interface
└── README.md              # This file



```
# Test API endpoints
ab -n 1000 -c 10 http://localhost:8080/api/v1/health
```

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

### Redis Monitoring
```bash
# Connect to Redis CLI
redis-cli

# Monitor Redis operations
MONITOR

# Check memory usage
INFO memory

# List all keys
KEYS *
```





