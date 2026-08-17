package main

import (
	"cepuin_chat/internal/websocket"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ws", websocket.Handler)

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
