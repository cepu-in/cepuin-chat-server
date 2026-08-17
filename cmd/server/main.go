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
		log.Fatal("Database connection failed: ", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected successfully")

	// =========================
	// REPOSITORY
	// =========================

	chatRepository := repository.NewChatRepository(db)
	chatHandler := chat.NewHandler(chatRepository)

	// =========================
	// WEBSOCKET
	// =========================

	wsHandler := websocket.NewHandler(chatRepository)

	// =========================
	// HTTP ROUTER
	// =========================

	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

	mux.HandleFunc("/ws", wsHandler.Handle)

	mux.HandleFunc(
		"/chat/history",
		chatHandler.GetHistory,
	)

	mux.HandleFunc(
		"/chat/list",
		chatHandler.GetList,
	)

	// =========================
	// SERVER
	// =========================

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("=====================CEPUIN CHAT SERVER starting on :8080 =======================")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(`{"status":"ok","service":"cepuin-chat"}`))
}
