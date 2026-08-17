package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"nhooyr.io/websocket"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Connecting to ws://localhost:8080/ws...")

	conn, _, err := websocket.Dial(ctx, "ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatalf("WebSocket connection failed: %v", err)
	}

	defer conn.Close(websocket.StatusNormalClosure, "")

	log.Println("Connected!")

	message := "Hello CEPUIN WebSocket!"

	log.Printf("Sending: %s", message)

	err = conn.Write(ctx, websocket.MessageText, []byte(message))
	if err != nil {
		log.Fatalf("Write failed: %v", err)
	}

	messageType, data, err := conn.Read(ctx)
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}

	fmt.Printf("Received: %s\n", string(data))

	if messageType == websocket.MessageText && string(data) == message {
		log.Println("========================================")
		log.Println("WEBSOCKET TEST SUCCESS")
		log.Println("========================================")
	} else {
		log.Println("WebSocket response does not match")
	}
}
