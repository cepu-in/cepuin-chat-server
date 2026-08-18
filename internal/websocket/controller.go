package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"cepuin_chat/internal/chat"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

type Controller struct {
	Service *chat.Service
	Manager *ClientManager
}

func NewController(
	service *chat.Service,
	manager *ClientManager,
) *Controller {
	return &Controller{
		Service: service,
		Manager: manager,
	}
}

func (c *Controller) Handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})

	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	log.Println("--------------WebSocket client connected---------------")

	ctx := r.Context()

	var currentUserID string

	defer func() {
		if currentUserID != "" {
			c.Manager.RemoveClient(currentUserID)
		}

		conn.Close(websocket.StatusNormalClosure, "")

		log.Println("--------------WebSocket client disconnected---------------")
	}()

	for {
		_, data, err := conn.Read(ctx)

		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		log.Printf("received message: %s", string(data))

		var msg ChatMessage

		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("invalid message JSON: %v", err)
			continue
		}

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

		// Register koneksi user.
		currentUserID = senderID.String()

		c.Manager.AddClient(
			currentUserID,
			conn,
		)

		log.Printf(
			"WebSocket user registered: %s",
			currentUserID,
		)

		// Business logic tetap di Service.
		savedMessage, err := c.Service.SendMessage(
			ctx,
			senderID,
			receiverID,
			msg.Message,
		)

		if err != nil {
			log.Printf("failed to send message: %v", err)
			continue
		}

		log.Printf(
			"message saved: message_id=%s conversation_id=%s",
			savedMessage.ID,
			savedMessage.ConversationID,
		)

		dataToSend, err := json.Marshal(savedMessage)

		if err != nil {
			log.Printf("failed to marshal message: %v", err)
			continue
		}

		err = c.Manager.SendToUser(
			context.Background(),
			receiverID.String(),
			dataToSend,
		)

		if err != nil {
			log.Printf(
				"failed to send message to receiver: %v",
				err,
			)
		}
	}
}
