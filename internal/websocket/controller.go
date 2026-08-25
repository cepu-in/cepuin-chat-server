package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cepuin_chat/internal/chat"
	"cepuin_chat/internal/storage"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

type Controller struct {
	Service *chat.Service
	Manager *Manager
	Storage *storage.WasabiStorage
}

func NewController(
	service *chat.Service,
	manager *Manager,
	storageService *storage.WasabiStorage,
) *Controller {
	return &Controller{
		Service: service,
		Manager: manager,
		Storage: storageService,
	}
}

func (c *Controller) Handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	conn, err := websocket.Accept(
		w,
		r,
		&websocket.AcceptOptions{
			InsecureSkipVerify: true,
		},
	)

	if err != nil {
		log.Printf(
			"websocket accept error: %v",
			err,
		)
		return
	}

	log.Println(
		"--------------WebSocket client connected---------------",
	)

	ctx := r.Context()

	// ============================================================
	// GET USER ID
	// ============================================================

	currentUserID := r.URL.Query().Get("user_id")

	if currentUserID == "" {
		log.Println(
			"WebSocket rejected: user_id is empty",
		)

		conn.Close(
			websocket.StatusPolicyViolation,
			"user_id required",
		)

		return
	}

	userID, err := uuid.Parse(currentUserID)

	if err != nil {
		log.Printf(
			"invalid user_id: %v",
			err,
		)

		conn.Close(
			websocket.StatusPolicyViolation,
			"invalid user_id",
		)

		return
	}

	currentUserID = userID.String()

	// ============================================================
	// REGISTER USER
	// ============================================================

	c.Manager.Add(
		currentUserID,
		conn,
	)

	log.Printf(
		"WebSocket user registered: %s",
		currentUserID,
	)

	// ============================================================
	// CLEANUP
	// ============================================================

	defer func() {

		c.Manager.Remove(
			currentUserID,
		)

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

	// ============================================================
	// READ MESSAGE
	// ============================================================

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

		if err := json.Unmarshal(
			data,
			&msg,
		); err != nil {

			log.Printf(
				"invalid message JSON: %v",
				err,
			)

			continue
		}

		// ========================================================
		// ROUTE MESSAGE BASED ON TYPE
		// ========================================================

		switch msg.Type {

		// ========================================================
		// NORMAL CHAT MESSAGE
		// ========================================================

		case "message":

			c.handleMessage(
				ctx,
				userID,
				msg,
			)

		// ========================================================
		// MARK CHAT AS READ
		// ========================================================

		case "mark_read":

			c.handleMarkRead(
				ctx,
				userID,
				msg,
			)

		// ========================================================
		// UNKNOWN TYPE
		// ========================================================

		default:

			log.Printf(
				"unknown websocket message type: %s",
				msg.Type,
			)
		}
	}
}

// ================================================================
// HANDLE NORMAL MESSAGE
// ================================================================

func (c *Controller) handleMessage(
	ctx context.Context,
	userID uuid.UUID,
	msg ChatMessage,
) {

	// ============================================================
	// VALIDATE RECEIVER
	// ============================================================

	receiverID, err := uuid.Parse(
		msg.ReceiverID,
	)

	if err != nil {

		log.Printf(
			"invalid receiver_id: %v",
			err,
		)

		return
	}

	// ============================================================
	// SAVE MESSAGE
	// ============================================================

	savedMessage, err := c.Service.SendMessage(
		ctx,
		userID,
		receiverID,
		msg.Message,
		msg.ImageKey,
	)

	if err != nil {

		log.Printf(
			"failed to send message: %v",
			err,
		)

		return
	}

	log.Printf(
		"message saved: message_id=%s conversation_id=%s",
		savedMessage.ID,
		savedMessage.ConversationID,
	)

	imageURL := ""

	if savedMessage.ImageKey != nil && *savedMessage.ImageKey != "" {

		if c.Storage == nil {
			log.Println("failed to generate image URL: storage is nil")
			return
		}

		imageURL, err = c.Storage.GetPresignedURL(
			ctx,
			*savedMessage.ImageKey,
			3600,
		)

		if err != nil {
			log.Printf(
				"failed to generate image presigned URL: key=%s error=%v",
				*savedMessage.ImageKey,
				err,
			)
			return
		}

		log.Printf(
			"IMAGE PRESIGNED URL GENERATED: %s",
			imageURL,
		)
	}

	// ============================================================
	// SERIALIZE MESSAGE
	// ============================================================

	dataToSend, err := json.Marshal(
		map[string]interface{}{
			"type":            "message",
			"id":              savedMessage.ID,
			"conversation_id": savedMessage.ConversationID,
			"sender_id":       savedMessage.SenderID,
			"receiver_id":     savedMessage.ReceiverID,
			"message":         savedMessage.Message,
			"image_key":       imageURL,
			"created_at":      savedMessage.CreatedAt,
			"read_at":         savedMessage.ReadAt,
		},
	)

	if err != nil {

		log.Printf(
			"failed to marshal message: %v",
			err,
		)

		return
	}

	// ============================================================
	// SEND TO RECEIVER
	// ============================================================

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

	notificationEvent := map[string]interface{}{
		"type":            "chat_notification",
		"conversation_id": savedMessage.ConversationID.String(),
		"sender_id":       savedMessage.SenderID.String(),
		"message_id":      savedMessage.ID.String(),
	}

	notificationData, err := json.Marshal(
		notificationEvent,
	)

	if err != nil {
		log.Printf(
			"failed to marshal chat notification: %v",
			err,
		)

		return
	}

	err = c.Manager.SendToUser(
		context.Background(),
		receiverID.String(),
		notificationData,
	)

	if err != nil {
		log.Printf(
			"failed to send chat notification: %v",
			err,
		)
	}
	// ============================================================
	// SEND BACK TO SENDER
	// ============================================================

	err = c.Manager.SendToUser(
		context.Background(),
		userID.String(),
		dataToSend,
	)

	if err != nil {

		log.Printf(
			"failed to send message back to sender: %v",
			err,
		)
	}
}

// ================================================================
// HANDLE MARK READ
// ================================================================

func (c *Controller) handleMarkRead(
	ctx context.Context,
	userID uuid.UUID,
	msg ChatMessage,
) {

	targetUserID, err := uuid.Parse(
		msg.ReceiverID,
	)

	if err != nil {

		log.Printf(
			"invalid mark_read receiver_id: %v",
			err,
		)

		return
	}

	// ============================================================
	// MARK MESSAGES AS READ
	// ============================================================

	updatedMessages, err := c.Service.MarkChatAsRead(
		ctx,
		userID,
		targetUserID,
	)

	if err != nil {

		log.Printf(
			"failed to mark messages as read: %v",
			err,
		)

		return
	}

	// Tidak ada message baru yang berubah menjadi read.
	if len(updatedMessages) == 0 {
		return
	}

	// ============================================================
	// AMBIL CONVERSATION
	// ============================================================

	conversationID := updatedMessages[0].ConversationID

	// ============================================================
	// AMBIL MESSAGE IDS
	// ============================================================

	messageIDs := make(
		[]string,
		0,
		len(updatedMessages),
	)

	var readAt *time.Time

	for _, message := range updatedMessages {

		messageIDs = append(
			messageIDs,
			message.ID.String(),
		)

		if message.ReadAt != nil {
			readAt = message.ReadAt
		}
	}

	if readAt == nil {
		log.Println(
			"mark_read succeeded but read_at is nil",
		)

		return
	}

	// ============================================================
	// CREATE READ RECEIPT EVENT
	// ============================================================

	event := map[string]interface{}{
		"type":            "message_read",
		"conversation_id": conversationID.String(),
		"message_ids":     messageIDs,
		"read_at":         readAt.UTC().Format(time.RFC3339),
	}

	dataToSend, err := json.Marshal(
		event,
	)

	if err != nil {

		log.Printf(
			"failed to marshal message_read event: %v",
			err,
		)

		return
	}

	// ============================================================
	// SEND READ RECEIPT TO ORIGINAL SENDER
	// ============================================================

	err = c.Manager.SendToUser(
		context.Background(),
		targetUserID.String(),
		dataToSend,
	)

	if err != nil {

		log.Printf(
			"failed to send message_read to sender: %v",
			err,
		)

		return
	}

	log.Printf(
		"message_read sent: user=%s target=%s messages=%d",
		userID,
		targetUserID,
		len(messageIDs),
	)
}
