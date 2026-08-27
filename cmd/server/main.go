package main

import (
	"cepuin_chat/internal/chat"
	"cepuin_chat/internal/database"
	"cepuin_chat/internal/repository"
	"cepuin_chat/internal/storage"
	"cepuin_chat/internal/websocket"
	"log"
	"net/http"

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
	// IMAGE SERVICE
	// =========================
	wasabiStorage, err := storage.NewWasabiStorage()

	if err != nil {
		log.Fatal(
			"failed to initialize Wasabi storage:",
			err,
		)
	}

	log.Println("Wasabi storage connected successfully")
	// =========================
	// REPOSITORY
	// =========================

	chatRepository := repository.NewChatRepository(db)

	// =========================
	// CHAT SERVICE
	// =========================

	chatService := chat.NewService(chatRepository, wasabiStorage)

	// =========================
	// CHAT CONTROLLER
	// =========================

	chatController := chat.NewController(chatService,
		wasabiStorage)

	// =========================
	// WEBSOCKET
	// =========================

	manager := websocket.NewManager()

	wsController := websocket.NewController(
		chatService,
		manager,
		wasabiStorage,
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
		Addr:    ":8081",
		Handler: mux,
	}

	log.Println(
		"=====================CEPUIN CHAT SERVER starting on :8081 =======================",
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
