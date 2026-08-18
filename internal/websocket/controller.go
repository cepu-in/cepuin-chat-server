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

	// =========================
	// GET USER ID
	// =========================

	currentUserID := r.URL.Query().Get("user_id")

	if currentUserID == "" {
		log.Println("WebSocket rejected: user_id is empty")

		conn.Close(
			websocket.StatusPolicyViolation,
			"user_id required",
		)

		return
	}

	userID, err := uuid.Parse(currentUserID)

	if err != nil {
		log.Printf("invalid user_id: %v", err)

		conn.Close(
			websocket.StatusPolicyViolation,
			"invalid user_id",
		)

		return
	}

	currentUserID = userID.String()

	// =========================
	// REGISTER USER IMMEDIATELY
	// =========================

	c.Manager.AddClient(
		currentUserID,
		conn,
	)

	log.Printf(
		"WebSocket user registered: %s",
		currentUserID,
	)

	// =========================
	// CLEANUP
	// =========================

	defer func() {
		c.Manager.RemoveClient(currentUserID)

		conn.Close(
			websocket.StatusNormalClosure,
			"",
		)

		log.Printf(
			"WebSocket user disconnected: %s",
			currentUserID,
		)

		log.Println(
			"--------------WebSocket client disconnected---------------",
		)
	}()

	// =========================
	// READ MESSAGE
	// =========================

	for {
		_, data, err := conn.Read(ctx)

		if err != nil {
			log.Printf(
				"WebSocket read error: %v",
				err,
			)

			return
		}

		log.Printf(
			"received message: %s",
			string(data),
		)

		var msg ChatMessage

		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf(
				"invalid message JSON: %v",
				err,
			)

			continue
		}

		// =========================
		// RECEIVER
		// =========================

		receiverID, err := uuid.Parse(msg.ReceiverID)

		if err != nil {
			log.Printf(
				"invalid receiver_id: %v",
				err,
			)

			continue
		}

		// =========================
		// SAVE MESSAGE
		// =========================

		savedMessage, err := c.Service.SendMessage(
			ctx,
			userID,
			receiverID,
			msg.Message,
		)

		if err != nil {
			log.Printf(
				"failed to send message: %v",
				err,
			)

			continue
		}

		log.Printf(
			"message saved: message_id=%s conversation_id=%s",
			savedMessage.ID,
			savedMessage.ConversationID,
		)

		// =========================
		// SERIALIZE MESSAGE
		// =========================

		dataToSend, err := json.Marshal(savedMessage)

		if err != nil {
			log.Printf(
				"failed to marshal message: %v",
				err,
			)

			continue
		}

		// =========================
		// SEND TO RECEIVER
		// =========================

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
