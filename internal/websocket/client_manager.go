package websocket

import (
	"context"
	"fmt"
	"log"
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

func (m *ClientManager) AddClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[userID] = conn

	log.Printf(
		"[WS ADD CLIENT] user=%s conn=%p total=%d",
		userID,
		conn,
		len(m.clients),
	)
}

func (m *ClientManager) RemoveClient(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.clients[userID]

	delete(m.clients, userID)

	log.Printf(
		"[WS REMOVE CLIENT] user=%s exists=%v conn=%p total=%d",
		userID,
		exists,
		conn,
		len(m.clients),
	)
}

func (cm *ClientManager) GetClient(
	userID string,
) (*websocket.Conn, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, ok := cm.clients[userID]

	return conn, ok
}

func (m *ClientManager) SendToUser(
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

	if !exists || conn == nil {
		log.Printf(
			"[WS SEND FAILED] target user not connected: %s",
			userID,
		)

		return fmt.Errorf(
			"user %s not connected",
			userID,
		)
	}

	err := conn.Write(
		ctx,
		websocket.MessageText,
		message,
	)

	if err != nil {
		log.Printf(
			"[WS SEND ERROR] target=%s err=%v",
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
