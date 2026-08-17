package chat

import (
	"encoding/json"
	"net/http"

	"cepuin_chat/internal/repository"

	"github.com/google/uuid"
)

type Handler struct {
	Repository *repository.ChatRepository
}

func NewHandler(repo *repository.ChatRepository) *Handler {
	return &Handler{
		Repository: repo,
	}
}

// ============================================================
// GET CHAT LIST
// ============================================================
// GET /chat/list?user_id=xxxx
func (h *Handler) GetList(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	userIDString := r.URL.Query().Get("user_id")

	if userIDString == "" {
		http.Error(
			w,
			`{"error":"user_id is required"}`,
			http.StatusBadRequest,
		)
		return
	}

	userID, err := uuid.Parse(userIDString)

	if err != nil {
		http.Error(
			w,
			`{"error":"invalid user_id"}`,
			http.StatusBadRequest,
		)
		return
	}

	chats, err := h.Repository.GetChatList(
		r.Context(),
		userID,
	)

	if err != nil {
		http.Error(
			w,
			`{"error":"failed to get chat list"}`,
			http.StatusInternalServerError,
		)
		return
	}

	if chats == nil {
		chats = []repository.ChatListItem{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"chats": chats,
	})
}

// ============================================================
// GET CHAT HISTORY
// ============================================================
// GET /chat/history?user_id=xxxx&target_user_id=xxxx
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	userIDString := r.URL.Query().Get("user_id")
	targetUserIDString := r.URL.Query().Get("target_user_id")

	if userIDString == "" || targetUserIDString == "" {
		http.Error(
			w,
			`{"error":"user_id and target_user_id are required"}`,
			http.StatusBadRequest,
		)
		return
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		http.Error(
			w,
			`{"error":"invalid user_id"}`,
			http.StatusBadRequest,
		)
		return
	}

	targetUserID, err := uuid.Parse(targetUserIDString)
	if err != nil {
		http.Error(
			w,
			`{"error":"invalid target_user_id"}`,
			http.StatusBadRequest,
		)
		return
	}

	messages, err := h.Repository.GetMessageHistory(
		r.Context(),
		userID,
		targetUserID,
	)

	if err != nil {
		http.Error(
			w,
			`{"error":"failed to get chat history"}`,
			http.StatusInternalServerError,
		)
		return
	}

	if messages == nil {
		messages = []repository.Message{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}
