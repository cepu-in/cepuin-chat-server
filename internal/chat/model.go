package chat

type ChatListResponse struct {
	Chats interface{} `json:"chats"`
}

type ChatHistoryResponse struct {
	Messages interface{} `json:"messages"`
}

type ChatListRequest struct {
	UserID string `json:"user_id"`
}

type ChatHistoryRequest struct {
	UserID       string `json:"user_id"`
	TargetUserID string `json:"target_user_id"`
}

type MessageReadEvent struct {
	Type     string `json:"type"`
	ReaderID string `json:"reader_id"`
	SenderID string `json:"sender_id"`
	ReadAt   string `json:"read_at"`
}
