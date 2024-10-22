package websocket

import (
	"errors"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/ebpf"
	"github.com/danger-dream/ebpf-firewall/internal/interfaces"
	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
	"github.com/gorilla/websocket"
)

type WebSocketMessageHandler struct {
	configManager  *config.ConfigManager
	ebpfManager    interfaces.EBPFManager // 使用接口类型
	ws             interfaces.WebSocketServer
	ruleMatcher    interfaces.RuleMatcher // 使用接口类型
	summaryManager *ebpf.SummaryManager
}

func NewWebSocketMessageHandler(
	configManager *config.ConfigManager,
	ebpfManager interfaces.EBPFManager, // 使用接口类型
	ws interfaces.WebSocketServer,
	ruleMatcher interfaces.RuleMatcher, // 使用接口类型
	summaryManager *ebpf.SummaryManager,
) *WebSocketMessageHandler {
	return &WebSocketMessageHandler{
		configManager:  configManager,
		ebpfManager:    ebpfManager,
		ws:             ws,
		ruleMatcher:    ruleMatcher,
		summaryManager: summaryManager,
	}
}

func (h *WebSocketMessageHandler) Handle(conn *websocket.Conn, params *interfaces.WebSocketMessage, client *interfaces.WebSocketClient) {
	if params.ID == "" {
		return
	}
	var err error
	var data interface{}
	switch params.Action {
	case "ping":
		data = "pong"
	case "get_link_type":
		data = h.ebpfManager.GetLinkType()
	case "get_summary":
		data = h.summaryManager.GetSummary()
	case "get_rules":
		data = h.ruleMatcher.GetRules()
	case "set_rules":
		data, err = h.handleSetRules(params.Payload)
	case "get_match_list":
		data, err = h.handleGetMatchList(params.Payload)
	case "get_black_list":
		data = h.configManager.Config.Black
	case "change_black":
		data, err = h.handleChangeBlack(params.Payload)
	case "get_system_resource_usage":
		data, err = utils.GetSystemResourceUsage()
	case "change_broadcast_status":
		data, err = h.handleChangeBroadcastStatus(params.Payload, client)
	default:
		return
	}

	if err != nil {
		conn.WriteJSON(interfaces.WebSocketMessage{
			Action:  "callback-error",
			ID:      params.ID,
			Payload: err.Error(),
		})
		return
	}
	conn.WriteJSON(interfaces.WebSocketMessage{
		Action:  "callback",
		ID:      params.ID,
		Payload: data,
	})
}

// 设置规则
func (h *WebSocketMessageHandler) handleSetRules(payload interface{}) (interface{}, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}
	if data, ok := payload.([]types.Rule); ok {
		count := h.ruleMatcher.ParseRules(data)
		if count > 0 {
			// 更新配置规则
			h.configManager.Config.Rules = data
			// 保存配置文件
			if err := h.configManager.SaveConfig(); err != nil {
				return nil, err
			}
			return count, nil
		} else {
			return nil, errors.New("no rules parsed")
		}
	}
	return nil, errors.New("payload format error")
}

// 获取匹配列表
func (h *WebSocketMessageHandler) handleGetMatchList(payload interface{}) (interface{}, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}
	if data, ok := payload.(types.WebSocketMatchQueryPayload); ok {
		return h.summaryManager.GetMatchList(&data), nil
	}
	return nil, errors.New("payload format error")
}

// 修改黑名单
func (h *WebSocketMessageHandler) handleChangeBlack(payload interface{}) (interface{}, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}
	if data, ok := payload.(types.WebSocketChangeBlackListPayload); ok {
		h.configManager.UpdateBlackList(data)
		return true, nil
	}
	return nil, errors.New("payload format error")
}

// 修改广播状态
func (h *WebSocketMessageHandler) handleChangeBroadcastStatus(payload interface{}, client *interfaces.WebSocketClient) (interface{}, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}
	if data, ok := payload.(bool); ok {
		client.ReceiveBroadcast = data
		return true, nil
	}
	return nil, errors.New("payload format error")
}
