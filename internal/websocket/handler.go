package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"cepuin_chat/internal/repository"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

// ClientManager menyimpan semua client WebSocket yang sedang terhubung.
type ClientManager struct {
	sync.Mutex
	clients map[*websocket.Conn]bool
}

// Handler menangani WebSocket + database.
type Handler struct {
	Repository *repository.ChatRepository
	Manager    *ClientManager
}

var manager = ClientManager{
	clients: make(map[*websocket.Conn]bool),
}

// NewHandler membuat WebSocket handler baru.
func NewHandler(repo *repository.ChatRepository) *Handler {
	return &Handler{
		Repository: repo,
		Manager:    &manager,
	}
}

// Format pesan yang dikirim Flutter.
type ChatMessage struct {
	Type       string `json:"type"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	// Daftarkan client.
	h.Manager.Lock()
	h.Manager.clients[conn] = true
	h.Manager.Unlock()

	// Cleanup ketika koneksi ditutup.
	defer func() {
		h.Manager.Lock()
		delete(h.Manager.clients, conn)
		h.Manager.Unlock()

		conn.Close(websocket.StatusNormalClosure, "")

		log.Println("--------------WebSocket client disconnected---------------")
	}()

	log.Println("--------------WebSocket client connected---------------")

	ctx := r.Context()

	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		log.Printf("received message: %s", string(data))

		// =========================
		// PARSE JSON
		// =========================

		var msg ChatMessage

		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("invalid message JSON: %v", err)
			continue
		}

		// =========================
		// VALIDATE UUID
		// =========================

		senderID, err := uuid.Parse(msg.SenderID)
		if err != nil {
			log.Printf("invalid sender_id: %v", err)
			continue
		}

		receiverID, err := uuid.Parse(msg.ReceiverID)
		if err != nil {
			log.Printf("invalid receiver_id: %v", err)
			continue
		}

		if senderID == receiverID {
			log.Println("sender and receiver cannot be the same")
			continue
		}

		if msg.Message == "" {
			log.Println("message cannot be empty")
			continue
		}

		// =========================
		// DATABASE
		// =========================

		conversationID, err := h.Repository.GetOrCreateConversation(
			ctx,
			senderID,
			receiverID,
		)

		if err != nil {
			log.Printf(
				"failed to get/create conversation: %v",
				err,
			)
			continue
		}

		messageID, err := h.Repository.CreateMessage(
			ctx,
			conversationID,
			senderID,
			receiverID,
			msg.Message,
		)

		if err != nil {
			log.Printf(
				"failed to save message: %v",
				err,
			)
			continue
		}

		log.Printf(
			"message saved: message_id=%s conversation_id=%s",
			messageID,
			conversationID,
		)

		// =========================
		// BROADCAST
		// =========================

		h.Manager.Lock()

		for client := range h.Manager.clients {

			// Jangan kirim kembali ke pengirim.
			// Ini mempertahankan behavior sebelumnya
			// supaya pesan tidak muncul dua kali.
			if client == conn {
				continue
			}

			err := client.Write(
				context.Background(),
				messageType,
				data,
			)

			if err != nil {
				log.Printf(
					"WebSocket write error: %v",
					err,
				)

				client.Close(
					websocket.StatusAbnormalClosure,
					"",
				)

				delete(h.Manager.clients, client)
			}
		}

		h.Manager.Unlock()
	}
}
