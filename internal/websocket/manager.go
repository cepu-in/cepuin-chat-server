package websocket

import (
	"context"
	"sync"

	"nhooyr.io/websocket"
)

type Manager struct {
	mu      sync.RWMutex
	clients map[string]*websocket.Conn
}

func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*websocket.Conn),
	}
}

func (m *Manager) Add(
	userID string,
	conn *websocket.Conn,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[userID] = conn
}

func (m *Manager) Remove(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, userID)
}

func (m *Manager) SendToUser(
	ctx context.Context,
	userID string,
	message []byte,
) error {

	m.mu.RLock()
	conn, exists := m.clients[userID]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	return conn.Write(
		ctx,
		websocket.MessageText,
		message,
	)
}
