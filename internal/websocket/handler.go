package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
)

// Hub untuk menyimpan daftar klien yang terhubung secara aman
type ClientManager struct {
	sync.Mutex
	clients map[*websocket.Conn]bool
}

var manager = ClientManager{
	clients: make(map[*websocket.Conn]bool),
}

func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Berguna untuk development di emulator lokal
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	// Daftarkan klien ke manager
	manager.Lock()
	manager.clients[conn] = true
	manager.Unlock()

	// Pastikan koneksi dihapus dan ditutup saat fungsi selesai
	defer func() {
		manager.Lock()
		delete(manager.clients, conn)
		manager.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
		log.Println("--------------WebSocket client disconnected---------------")
	}()

	log.Println("--------------WebSocket client connected---------------")

	ctx := r.Context()

	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}

		log.Printf("received message: %s", string(data))

		// BROADCAST KE KLIEN LAIN (Kecuali pengirim sendiri)
		// Ini mencegah pesan muncul dua kali di emulator pengirim (duplikat kanan-kiri)
		manager.Lock()
		for client := range manager.clients {
			if client != conn { // Jangan kirim balik ke pengirim
				err = client.Write(context.Background(), messageType, data)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					client.Close(websocket.StatusAbnormalClosure, "")
					delete(manager.clients, client)
				}
			}
		}
		manager.Unlock()
	}
}
