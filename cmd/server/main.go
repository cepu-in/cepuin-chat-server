package main

import (
	"log"
	"net/http"

	"cepuin_chat/internal/chat"
	"cepuin_chat/internal/database"
	"cepuin_chat/internal/repository"
	"cepuin_chat/internal/websocket"

	"github.com/joho/godotenv"
)

func main() {
	// =========================
	// LOAD ENV
	// =========================

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// =========================
	// DATABASE
	// =========================

	db, err := database.NewPostgresPool()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected successfully")

	// =========================
	// REPOSITORY
	// =========================

	chatRepository := repository.NewChatRepository(db)

	// =========================
	// CHAT SERVICE
	// =========================

	chatService := chat.NewService(chatRepository)

	// =========================
	// CHAT CONTROLLER
	// =========================

	chatController := chat.NewController(chatService)

	// =========================
	// WEBSOCKET
	// =========================

	clientManager := websocket.NewClientManager()

	wsController := websocket.NewController(
		chatService,
		clientManager,
	)

	// =========================
	// HTTP ROUTER
	// =========================

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc(
		"/health",
		healthHandler,
	)

	// Chat HTTP routes
	chat.RegisterRoutes(
		mux,
		chatController,
	)

	// WebSocket
	mux.HandleFunc(
		"/ws",
		wsController.Handle,
	)

	// =========================
	// SERVER
	// =========================

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println(
		"=====================CEPUIN CHAT SERVER starting on :8080 =======================",
	)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(
		`{"status":"ok","service":"cepuin-chat"}`,
	))
}
