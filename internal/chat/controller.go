package chat

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type Controller struct {
	Service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{
		Service: service,
	}
}

// ============================================================
// GET CHAT LIST
// ============================================================

func (c *Controller) GetList(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

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

	chats, err := c.Service.GetChatList(
		r.Context(),
		userID,
	)

	if err != nil {
		log.Printf("GET CHAT LIST ERROR: %v", err)

		http.Error(
			w,
			`{"error":"failed to get chat list"}`,
			http.StatusInternalServerError,
		)
		return
	}

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"chats": chats,
		},
	)
}

// ============================================================
// GET CHAT HISTORY
// ============================================================

// ============================================================
// GET CHAT HISTORY
// ============================================================

func (c *Controller) GetHistory(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

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

	// ============================================================
	// PAGINATION
	// ============================================================

	limit := 20
	offset := 0

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)

		if err != nil || parsed <= 0 {
			http.Error(
				w,
				`{"error":"invalid limit"}`,
				http.StatusBadRequest,
			)
			return
		}

		limit = parsed
	}

	if value := r.URL.Query().Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)

		if err != nil || parsed < 0 {
			http.Error(
				w,
				`{"error":"invalid offset"}`,
				http.StatusBadRequest,
			)
			return
		}

		offset = parsed
	}

	// Batasi maksimal supaya client tidak meminta ribuan message
	if limit > 100 {
		limit = 100
	}

	// ============================================================
	// GET HISTORY
	// ============================================================

	messages, err := c.Service.GetChatHistory(
		r.Context(),
		userID,
		targetUserID,
		limit,
		offset,
	)

	if err != nil {
		log.Printf(
			"GET CHAT HISTORY ERROR: user=%s target=%s limit=%d offset=%d error=%v",
			userID,
			targetUserID,
			limit,
			offset,
			err,
		)

		http.Error(
			w,
			`{"error":"failed to get chat history"}`,
			http.StatusInternalServerError,
		)
		return
	}

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"messages": messages,
			"limit":    limit,
			"offset":   offset,
			"has_more": len(messages) == limit,
		},
	)
}
