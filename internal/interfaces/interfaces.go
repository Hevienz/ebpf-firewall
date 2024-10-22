package interfaces

import (
	http "net/http"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/gorilla/websocket"
)

type WebSocketServer interface {
	BroadcastSummary(summary interface{})
	BroadcastBlackEvent(key string)
	Start(port int)
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	Shutdown() error
}

type EBPFManager interface {
	Start() error
	Shutdown()
	GetLinkType() string
}

type WebSocketClient struct {
	ConnectTime      time.Time
	LastTime         time.Time
	ReceiveBroadcast bool
}

type WebSocketManager struct {
	Clients    map[*websocket.Conn]WebSocketClient
	Broadcast  chan WebSocketMessage
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
}

type WebSocketMessage struct {
	ID      string      `json:"id"`
	Action  string      `json:"action"`
	Payload interface{} `json:"payload"`
}

type WebSocketMessageHandler interface {
	Handle(conn *websocket.Conn, params *WebSocketMessage, client *WebSocketClient)
}

type RuleMatcher interface {
	MatchPacket(packet types.EnhancedPacketInfo) (bool, string)
	GetRules() []types.Rule
	ParseRules(rules []types.Rule) int
}
