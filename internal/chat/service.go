package chat

import (
	"context"
	"fmt"

	"cepuin_chat/internal/repository"

	"github.com/google/uuid"
)

type Service struct {
	Repository *repository.ChatRepository
}

func NewService(repo *repository.ChatRepository) *Service {
	return &Service{
		Repository: repo,
	}
}

// ============================================================
// GET CHAT LIST
// ============================================================

func (s *Service) GetChatList(
	ctx context.Context,
	userID uuid.UUID,
) ([]repository.ChatListItem, error) {

	chats, err := s.Repository.GetChatList(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service get chat list: %w", err)
	}

	if chats == nil {
		chats = []repository.ChatListItem{}
	}

	return chats, nil
}

// ============================================================
// GET CHAT HISTORY
// ============================================================

func (s *Service) GetChatHistory(
	ctx context.Context,
	userID uuid.UUID,
	targetUserID uuid.UUID,
	limit int,
	offset int,
) ([]repository.Message, error) {

	if userID == targetUserID {
		return []repository.Message{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	messages, err := s.Repository.GetMessageHistory(
		ctx,
		userID,
		targetUserID,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	if messages == nil {
		return []repository.Message{}, nil
	}

	return messages, nil
}

// ============================================================
// SEND MESSAGE
// ============================================================

func (s *Service) SendMessage(
	ctx context.Context,
	senderID uuid.UUID,
	receiverID uuid.UUID,
	message string,
) (*repository.Message, error) {

	if senderID == receiverID {
		return nil, fmt.Errorf("sender and receiver cannot be the same")
	}

	if message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	conversationID, err := s.Repository.GetOrCreateConversation(
		ctx,
		senderID,
		receiverID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get/create conversation: %w",
			err,
		)
	}

	messageID, err := s.Repository.CreateMessage(
		ctx,
		conversationID,
		senderID,
		receiverID,
		message,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create message: %w",
			err,
		)
	}

	return &repository.Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderID:       senderID,
		ReceiverID:     receiverID,
		Message:        message,
	}, nil
}

// ============================================================
// MARK CHAT AS READ
// ============================================================

func (s *Service) MarkChatAsRead(
	ctx context.Context,
	userID uuid.UUID,
	targetUserID uuid.UUID,
) (int64, error) {

	if userID == targetUserID {
		return 0, nil
	}

	updated, err := s.Repository.MarkMessagesAsRead(
		ctx,
		userID,
		targetUserID,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"service mark chat as read: %w",
			err,
		)
	}

	return updated, nil
}
