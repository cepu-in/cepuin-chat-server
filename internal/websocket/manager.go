package websocket

import (
	"context"
	"fmt"
	"log"
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

	log.Printf(
		"[WS SEND] target=%s exists=%v conn=%p",
		userID,
		exists,
		conn,
	)

	if !exists {
		log.Printf(
			"[WS SEND FAILED] target user not connected: %s",
			userID,
		)
		return fmt.Errorf("user %s not connected", userID)
	}

	err := conn.Write(
		ctx,
		websocket.MessageText,
		message,
	)

	if err != nil {
		log.Printf(
			"[WS SEND ERROR] target=%s error=%v",
			userID,
			err,
		)
		return err
	}

	log.Printf(
		"[WS SEND OK] target=%s",
		userID,
	)

	return nil
}
