package chat

import (
	"cepuin_chat/internal/storage"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Controller struct {
	Service *Service
	Storage *storage.WasabiStorage
}

func NewController(
	service *Service,
	storage *storage.WasabiStorage,
) *Controller {
	return &Controller{
		Service: service,
		Storage: storage,
	}
}

// ============================================================
// UPLOAD CHAT IMAGE
// ============================================================

// ============================================================
// UPLOAD CHAT IMAGE
// ============================================================

func (c *Controller) UploadImage(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	// ============================================================
	// METHOD
	// ============================================================

	if r.Method != http.MethodPost {
		http.Error(
			w,
			`{"error":"method not allowed"}`,
			http.StatusMethodNotAllowed,
		)
		return
	}

	// ============================================================
	// STORAGE CHECK
	// ============================================================

	if c.Storage == nil {
		log.Println("UPLOAD CHAT IMAGE ERROR: Wasabi storage is nil")

		http.Error(
			w,
			`{"error":"storage service is not available"}`,
			http.StatusInternalServerError,
		)
		return
	}

	// ============================================================
	// REQUEST LIMIT
	// ============================================================

	// Maksimal request 10 MB.
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		10<<20,
	)

	// ============================================================
	// MULTIPART
	// ============================================================

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf(
			"UPLOAD CHAT IMAGE MULTIPART ERROR: %v",
			err,
		)

		http.Error(
			w,
			`{"error":"invalid multipart form"}`,
			http.StatusBadRequest,
		)
		return
	}

	// ============================================================
	// RECEIVER ID
	// ============================================================

	receiverIDString := strings.TrimSpace(
		r.FormValue("receiver_id"),
	)

	if receiverIDString == "" {
		http.Error(
			w,
			`{"error":"receiver_id is required"}`,
			http.StatusBadRequest,
		)
		return
	}

	receiverID, err := uuid.Parse(receiverIDString)

	if err != nil {
		http.Error(
			w,
			`{"error":"invalid receiver_id"}`,
			http.StatusBadRequest,
		)
		return
	}

	// ============================================================
	// FILE
	// ============================================================

	file, fileHeader, err := r.FormFile("image")

	if err != nil {
		log.Printf(
			"UPLOAD CHAT IMAGE FORM FILE ERROR: %v",
			err,
		)

		http.Error(
			w,
			`{"error":"image file is required"}`,
			http.StatusBadRequest,
		)
		return
	}

	defer file.Close()

	// ============================================================
	// READ FILE
	// ============================================================

	data, err := io.ReadAll(file)

	if err != nil {
		log.Printf(
			"UPLOAD CHAT IMAGE READ ERROR: %v",
			err,
		)

		http.Error(
			w,
			`{"error":"failed to read image"}`,
			http.StatusInternalServerError,
		)
		return
	}

	if len(data) == 0 {
		http.Error(
			w,
			`{"error":"image file is empty"}`,
			http.StatusBadRequest,
		)
		return
	}

	// ============================================================
	// CONTENT TYPE
	// ============================================================

	// Jangan percaya Content-Type dari client.
	clientContentType := fileHeader.Header.Get("Content-Type")

	// Deteksi berdasarkan isi file.
	detectedContentType := http.DetectContentType(data)

	log.Printf(
		"CHAT IMAGE DEBUG: filename=%s size=%d client_content_type=%s detected_content_type=%s",
		fileHeader.Filename,
		len(data),
		clientContentType,
		detectedContentType,
	)

	// ============================================================
	// VALIDASI CONTENT TYPE
	// ============================================================

	contentType := detectedContentType

	// http.DetectContentType hanya mengenali beberapa format.
	// Kalau tidak dikenali, fallback berdasarkan ekstensi.
	if !strings.HasPrefix(
		strings.ToLower(contentType),
		"image/",
	) {
		ext := strings.ToLower(
			filepath.Ext(fileHeader.Filename),
		)

		switch ext {

		case ".jpg", ".jpeg":
			contentType = "image/jpeg"

		case ".png":
			contentType = "image/png"

		case ".webp":
			contentType = "image/webp"

		case ".gif":
			contentType = "image/gif"

		default:
			log.Printf(
				"CHAT IMAGE INVALID: filename=%s detected=%s extension=%s",
				fileHeader.Filename,
				detectedContentType,
				ext,
			)

			http.Error(
				w,
				`{"error":"file harus berupa gambar"}`,
				http.StatusBadRequest,
			)
			return
		}
	}

	// ============================================================
	// PASTIKAN FORMAT YANG DIDUKUNG
	// ============================================================

	switch strings.ToLower(contentType) {

	case "image/jpeg":
	case "image/png":
	case "image/webp":

	default:
		log.Printf(
			"CHAT IMAGE UNSUPPORTED: filename=%s content_type=%s",
			fileHeader.Filename,
			contentType,
		)

		http.Error(
			w,
			`{"error":"format gambar tidak didukung, gunakan JPG, PNG, atau WEBP"}`,
			http.StatusBadRequest,
		)
		return
	}

	// ============================================================
	// LOG
	// ============================================================

	log.Printf(
		"CHAT IMAGE UPLOAD: filename=%s size=%d content_type=%s receiver=%s",
		fileHeader.Filename,
		len(data),
		contentType,
		receiverID,
	)

	// ============================================================
	// UPLOAD WASABI
	// ============================================================

	key, err := c.Storage.UploadFile(
		r.Context(),
		data,
		fileHeader.Filename,
		contentType,
		"chat/images",
	)

	if err != nil {
		log.Printf(
			"UPLOAD CHAT IMAGE WASABI ERROR: %v",
			err,
		)

		http.Error(
			w,
			`{"error":"failed to upload image"}`,
			http.StatusInternalServerError,
		)
		return
	}

	log.Printf(
		"CHAT IMAGE UPLOAD SUCCESS: key=%s",
		key,
	)

	// ============================================================
	// GET PRESIGNED URL
	// ============================================================

	imageURL, err := c.Storage.GetPresignedURL(
		r.Context(),
		key,
		3600,
	)

	if err != nil {
		log.Printf(
			"CHAT IMAGE PRESIGN ERROR: key=%s error=%v",
			key,
			err,
		)

		http.Error(
			w,
			`{"error":"failed to generate image url"}`,
			http.StatusInternalServerError,
		)
		return
	}

	// ============================================================
	// RESPONSE
	// ============================================================

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(
		map[string]interface{}{
			"success": true,
			"message": "Image uploaded successfully",
			"data": map[string]interface{}{
				"image_key": key,
				"image_url": imageURL,
			},
		},
	)

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

// ============================================================
// MARK CHAT AS READ
// ============================================================

func (c *Controller) MarkAsRead(
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
	// MARK AS READ
	// ============================================================

	updatedMessages, err := c.Service.MarkChatAsRead(
		r.Context(),
		userID,
		targetUserID,
	)

	if err != nil {
		log.Printf(
			"MARK CHAT AS READ ERROR: user=%s target=%s error=%v",
			userID,
			targetUserID,
			err,
		)

		http.Error(
			w,
			`{"error":"failed to mark chat as read"}`,
			http.StatusInternalServerError,
		)
		return
	}

	// ============================================================
	// RESPONSE
	// ============================================================

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"success":       true,
			"updated_count": len(updatedMessages),
			"messages":      updatedMessages,
		},
	)
}
