package websocket

import (
	"context"
	"sync"

	"nhooyr.io/websocket"
)

type ClientManager struct {
	clients map[string]*websocket.Conn
	mu      sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*websocket.Conn),
	}
}

func (cm *ClientManager) AddClient(
	userID string,
	conn *websocket.Conn,
) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.clients[userID] = conn
}

func (cm *ClientManager) RemoveClient(
	userID string,
) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.clients, userID)
}

func (cm *ClientManager) GetClient(
	userID string,
) (*websocket.Conn, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, ok := cm.clients[userID]

	return conn, ok
}

func (cm *ClientManager) SendToUser(
	ctx context.Context,
	userID string,
	message []byte,
) error {

	cm.mu.RLock()
	conn, exists := cm.clients[userID]
	cm.mu.RUnlock()

	if !exists {
		// Receiver sedang offline.
		return nil
	}

	return conn.Write(
		ctx,
		websocket.MessageText,
		message,
	)
}
