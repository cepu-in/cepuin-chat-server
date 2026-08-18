package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	DB *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{
		DB: db,
	}
}

type ChatHistoryMessage struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	ReceiverID     uuid.UUID  `json:"receiver_id"`
	Message        string     `json:"message"`
	CreatedAt      time.Time  `json:"created_at"`
	ReadAt         *time.Time `json:"read_at"`
}

type ConversationItem struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	TargetUserID   uuid.UUID `json:"target_user_id"`
	LastMessage    string    `json:"last_message"`
	LastMessageAt  time.Time `json:"last_message_at"`
}

type ChatListItem struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	OtherUserID    uuid.UUID `json:"other_user_id"`
	LastMessage    string    `json:"last_message"`
	LastMessageAt  time.Time `json:"last_message_at"`
	UnreadCount    int       `json:"unread_count"`
}

type Message struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	ReceiverID     uuid.UUID  `json:"receiver_id"`
	Message        string     `json:"message"`
	CreatedAt      time.Time  `json:"created_at"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
}

// GetOrCreateConversation mencari conversation antara dua user.
// Urutan user_a_id dan user_b_id dibuat konsisten supaya
// conversation yang sama tidak dibuat dua kali.
func (r *ChatRepository) GetOrCreateConversation(
	ctx context.Context,
	userA uuid.UUID,
	userB uuid.UUID,
) (uuid.UUID, error) {

	if userA == userB {
		return uuid.Nil, fmt.Errorf("sender and receiver cannot be the same")
	}

	// Pastikan urutan UUID konsisten.
	if userB.String() < userA.String() {
		userA, userB = userB, userA
	}

	var conversationID uuid.UUID

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT id
		FROM conversations
		WHERE user_a_id = $1
		  AND user_b_id = $2
		LIMIT 1
		`,
		userA,
		userB,
	).Scan(&conversationID)

	if err == nil {
		return conversationID, nil
	}

	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	conversationID = uuid.New()

	_, err = r.DB.Exec(
		ctx,
		`
		INSERT INTO conversations (
			id,
			user_a_id,
			user_b_id,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			NOW(),
			NOW()
		)
		`,
		conversationID,
		userA,
		userB,
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversationID, nil
}

func (r *ChatRepository) GetChatList(
	ctx context.Context,
	userID uuid.UUID,
) ([]ChatListItem, error) {

	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			c.id AS conversation_id,

			CASE
				WHEN c.user_a_id = $1 THEN c.user_b_id
				ELSE c.user_a_id
			END AS other_user_id,

			m.message AS last_message,
			m.created_at AS last_message_at,

			COUNT(unread.id) AS unread_count

		FROM conversations c

		LEFT JOIN LATERAL (
			SELECT
				message,
				created_at
			FROM messages
			WHERE conversation_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true

		LEFT JOIN messages unread
			ON unread.conversation_id = c.id
			AND unread.receiver_id = $1
			AND unread.read_at IS NULL

		WHERE c.user_a_id = $1
		   OR c.user_b_id = $1

		GROUP BY
			c.id,
			c.user_a_id,
			c.user_b_id,
			m.message,
			m.created_at

		ORDER BY
			m.created_at DESC
		`,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get chat list: %w",
			err,
		)
	}

	defer rows.Close()

	var chats []ChatListItem

	for rows.Next() {
		var item ChatListItem

		err := rows.Scan(
			&item.ConversationID,
			&item.OtherUserID,
			&item.LastMessage,
			&item.LastMessageAt,
			&item.UnreadCount,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan chat list: %w",
				err,
			)
		}

		chats = append(chats, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"chat list rows error: %w",
			err,
		)
	}

	return chats, nil
}

func (r *ChatRepository) GetConversations(
	ctx context.Context,
	userID uuid.UUID,
) ([]ConversationItem, error) {

	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			c.id AS conversation_id,

			CASE
				WHEN c.user_a_id = $1 THEN c.user_b_id
				ELSE c.user_a_id
			END AS target_user_id,

			m.message AS last_message,
			m.created_at AS last_message_at

		FROM conversations c

		LEFT JOIN LATERAL (
			SELECT
				message,
				created_at
			FROM messages
			WHERE conversation_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true

		WHERE c.user_a_id = $1
		   OR c.user_b_id = $1

		ORDER BY
			COALESCE(m.created_at, c.updated_at) DESC
		`,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	defer rows.Close()

	var conversations []ConversationItem

	for rows.Next() {
		var item ConversationItem

		err := rows.Scan(
			&item.ConversationID,
			&item.TargetUserID,
			&item.LastMessage,
			&item.LastMessageAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan conversation: %w",
				err,
			)
		}

		conversations = append(conversations, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"conversation rows error: %w",
			err,
		)
	}

	return conversations, nil
}

func (r *ChatRepository) CreateMessage(
	ctx context.Context,
	conversationID uuid.UUID,
	senderID uuid.UUID,
	receiverID uuid.UUID,
	message string,
) (uuid.UUID, error) {

	messageID := uuid.New()

	_, err := r.DB.Exec(
		ctx,
		`
		INSERT INTO messages (
			id,
			conversation_id,
			sender_id,
			receiver_id,
			message,
			created_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			NOW()
		)
		`,
		messageID,
		conversationID,
		senderID,
		receiverID,
		message,
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create message: %w", err)
	}

	return messageID, nil
}

func (r *ChatRepository) GetChatHistory(
	ctx context.Context,
	userA uuid.UUID,
	userB uuid.UUID,
) ([]ChatHistoryMessage, error) {

	if userA == userB {
		return nil, fmt.Errorf("sender and receiver cannot be the same")
	}

	// Urutan UUID harus sama dengan GetOrCreateConversation.
	if userB.String() < userA.String() {
		userA, userB = userB, userA
	}

	var conversationID uuid.UUID

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT id
		FROM conversations
		WHERE user_a_id = $1
		  AND user_b_id = $2
		LIMIT 1
		`,
		userA,
		userB,
	).Scan(&conversationID)

	if err == pgx.ErrNoRows {
		// Belum pernah chat.
		return []ChatHistoryMessage{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"failed to find conversation: %w",
			err,
		)
	}

	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			id,
			conversation_id,
			sender_id,
			receiver_id,
			message,
			created_at,
			read_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		`,
		conversationID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get chat history: %w",
			err,
		)
	}

	defer rows.Close()

	messages := make([]ChatHistoryMessage, 0)

	for rows.Next() {
		var msg ChatHistoryMessage

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SenderID,
			&msg.ReceiverID,
			&msg.Message,
			&msg.CreatedAt,
			&msg.ReadAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan chat history: %w",
				err,
			)
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed to iterate chat history: %w",
			err,
		)
	}

	return messages, nil
}

func (r *ChatRepository) GetMessageHistory(
	ctx context.Context,
	userID uuid.UUID,
	targetUserID uuid.UUID,
) ([]Message, error) {

	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			m.id,
			m.conversation_id,
			m.sender_id,
			m.receiver_id,
			m.message,
			m.created_at,
			m.read_at
		FROM messages m
		INNER JOIN conversations c
			ON c.id = m.conversation_id
		WHERE
			(
				c.user_a_id = $1
				AND c.user_b_id = $2
			)
			OR
			(
				c.user_a_id = $2
				AND c.user_b_id = $1
			)
		ORDER BY m.created_at ASC
		`,
		userID,
		targetUserID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	defer rows.Close()

	var messages []Message

	for rows.Next() {

		var message Message

		err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.ReceiverID,
			&message.Message,
			&message.CreatedAt,
			&message.ReadAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("message rows error: %w", err)
	}

	return messages, nil
}
