package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/interfaces"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketServer struct {
	manager         *interfaces.WebSocketManager
	messageHandler  interfaces.WebSocketMessageHandler
	mutex           sync.Mutex
	mux             *http.ServeMux
	server          *http.Server
	shutdownChannel chan struct{}
	staticFiles     http.FileSystem
}

func NewWebSocketServer(fs http.FileSystem) *WebSocketServer {
	manager := &interfaces.WebSocketManager{
		Clients:    make(map[*websocket.Conn]interfaces.WebSocketClient),
		Broadcast:  make(chan interfaces.WebSocketMessage),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
	}

	return &WebSocketServer{
		manager:         manager,
		mux:             http.NewServeMux(),
		shutdownChannel: make(chan struct{}),
		staticFiles:     fs,
	}
}

func (s *WebSocketServer) Start(addr string) {
	go s.run()

	s.mux.HandleFunc("/ws", s.HandleWebSocket)

	if s.staticFiles != nil {
		s.mux.Handle("/", http.FileServer(s.staticFiles))
	}

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	log.Printf("Starting WebSocket server on addr %s", addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start WebSocket server: %v", err)
	}
}

func (s *WebSocketServer) run() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-s.shutdownChannel:
			ticker.Stop()
			log.Println("WebSocket server shutdown")
			return
		case client := <-s.manager.Register:
			s.mutex.Lock()
			s.manager.Clients[client] = interfaces.WebSocketClient{
				ConnectTime:      time.Now(),
				LastTime:         time.Now(),
				ReceiveBroadcast: true,
			}
			s.mutex.Unlock()
		case client := <-s.manager.Unregister:
			s.mutex.Lock()
			if _, ok := s.manager.Clients[client]; ok {
				delete(s.manager.Clients, client)
				client.Close()
			}
			s.mutex.Unlock()
		case message := <-s.manager.Broadcast:
			s.mutex.Lock()
			for conn, client := range s.manager.Clients {
				if client.ReceiveBroadcast {
					err := conn.WriteJSON(message)
					if err != nil {
						log.Printf("error: %v", err)
						conn.Close()
						delete(s.manager.Clients, conn)
					}
				}
			}
			s.mutex.Unlock()
		case <-ticker.C:
			s.mutex.Lock()
			for conn, client := range s.manager.Clients {
				if time.Since(client.LastTime) > 60*time.Second {
					delete(s.manager.Clients, conn)
					conn.Close()
				}
			}
			s.mutex.Unlock()
		}

	}
}

func (s *WebSocketServer) SetMessageHandler(handler interfaces.WebSocketMessageHandler) {
	s.messageHandler = handler
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	s.manager.Register <- conn

	go s.readPump(conn)
}

func (s *WebSocketServer) readPump(conn *websocket.Conn) {
	defer func() {
		s.manager.Unregister <- conn
		conn.Close()
	}()
	client := s.manager.Clients[conn]

	errNum := 0
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				break
			}
			errNum++
			if errNum > 10 {
				break
			}
			continue
		}
		errNum = 0
		client.LastTime = time.Now()

		var msg interfaces.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshalling message: %v", err)
			continue
		}
		s.messageHandler.Handle(conn, &msg, &client)
	}
}

func (s *WebSocketServer) BroadcastSummary(summary interface{}) {
	message := interfaces.WebSocketMessage{
		Action:  "broadcast-summary",
		Payload: summary,
	}
	s.manager.Broadcast <- message
}

func (s *WebSocketServer) BroadcastBlackEvent(key string) {
	message := interfaces.WebSocketMessage{
		Action:  "broadcast-black",
		Payload: key,
	}
	s.manager.Broadcast <- message
}

func (s *WebSocketServer) Shutdown() error {
	log.Println("Shutting down WebSocket server...")

	s.shutdownChannel <- struct{}{}
	s.mutex.Lock()
	for client := range s.manager.Clients {
		client.Close()
	}
	s.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("WebSocket server shutdown failed: %v", err)
	}

	log.Println("WebSocket server stopped")
	return nil
}
