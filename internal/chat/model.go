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
