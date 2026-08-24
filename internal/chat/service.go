package chat

import (
	"context"
	"fmt"

	"cepuin_chat/internal/repository"
	"cepuin_chat/internal/storage"

	"github.com/google/uuid"
)

type Service struct {
	Repository *repository.ChatRepository
	Storage    *storage.WasabiStorage
}

func NewService(
	repo *repository.ChatRepository,
	storageService *storage.WasabiStorage,
) *Service {
	return &Service{
		Repository: repo,
		Storage:    storageService,
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

	// ============================================================
	// GENERATE PRESIGNED URL UNTUK IMAGE
	// ============================================================

	for i := range messages {

		if messages[i].ImageKey == nil {
			continue
		}

		imageKey := *messages[i].ImageKey

		if imageKey == "" {
			continue
		}

		if s.Storage == nil {
			return nil, fmt.Errorf(
				"storage service is nil",
			)
		}

		imageURL, err := s.Storage.GetPresignedURL(
			ctx,
			imageKey,
			3600,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to generate image URL for key %s: %w",
				imageKey,
				err,
			)
		}

		messages[i].ImageKey = &imageURL
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
	imageKey *string,
) (*repository.Message, error) {

	if senderID == receiverID {
		return nil, fmt.Errorf(
			"sender and receiver cannot be the same",
		)
	}

	// Minimal harus ada message atau image
	if message == "" &&
		(imageKey == nil || *imageKey == "") {
		return nil, fmt.Errorf(
			"message or image is required",
		)
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

	savedMessage, err := s.Repository.CreateMessage(
		ctx,
		conversationID,
		senderID,
		receiverID,
		message,
		imageKey,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create message: %w",
			err,
		)
	}

	return savedMessage, nil
}

// ============================================================
// MARK CHAT AS READ
// ============================================================

func (s *Service) MarkChatAsRead(
	ctx context.Context,
	userID uuid.UUID,
	targetUserID uuid.UUID,
) ([]repository.Message, error) {

	if userID == targetUserID {
		return []repository.Message{}, nil
	}

	updatedMessages, err := s.Repository.MarkMessagesAsRead(
		ctx,
		userID,
		targetUserID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"service mark chat as read: %w",
			err,
		)
	}

	if updatedMessages == nil {
		updatedMessages = []repository.Message{}
	}

	return updatedMessages, nil
}

// ============================================================
// SEND IMAGE MESSAGE
// ============================================================

func (s *Service) SendImageMessage(
	ctx context.Context,
	senderID uuid.UUID,
	receiverID uuid.UUID,
	imageKey string,
) (*repository.Message, error) {

	if senderID == receiverID {
		return nil, fmt.Errorf(
			"sender and receiver cannot be the same",
		)
	}

	if imageKey == "" {
		return nil, fmt.Errorf(
			"image key is required",
		)
	}

	// Image tetap dianggap sebagai message biasa.
	// message = ""
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

	savedMessage, err := s.Repository.CreateMessage(
		ctx,
		conversationID,
		senderID,
		receiverID,
		"",
		&imageKey,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create image message: %w",
			err,
		)
	}

	return savedMessage, nil
}
