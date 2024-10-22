package ebpf

import (
	"net"
	"sort"
	"sync"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/interfaces"
	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
	"github.com/oschwald/geoip2-golang"
)

type SummaryManager struct {
	configManager   *config.ConfigManager
	ruleMatcher     interfaces.RuleMatcher
	wsServer        interfaces.WebSocketServer
	geoipDB         *geoip2.Reader
	summaryTicker   *time.Ticker
	shutdownChannel chan struct{}
	packetChan      chan types.EnhancedPacketInfo
	blackChan       chan types.BlackEvent
	cachePacket     []types.EnhancedPacketInfo
	mutex           sync.Mutex
	countrySummary  map[string]types.Summary
	citySummary     map[string]types.Summary
	ethTypeSummary  map[string]types.Summary
	ipProtoSummary  map[string]types.Summary
	daySummary      map[string]types.Summary
	matchSummary    map[string]types.Summary
	dstPortSummary  map[uint16]types.Summary
	inputPackets    map[string]types.InputPacket
	matchPackets    map[string][]types.EnhancedPacketInfo
	blackSummary    map[string]uint64
}

func NewSummary(configManager *config.ConfigManager, ruleMatcher interfaces.RuleMatcher, wsServer interfaces.WebSocketServer, geoipDB *geoip2.Reader) *SummaryManager {
	sm := &SummaryManager{
		configManager:   configManager,
		ruleMatcher:     ruleMatcher,
		wsServer:        wsServer,
		geoipDB:         geoipDB,
		summaryTicker:   time.NewTicker(time.Duration(configManager.Config.SummaryTime) * time.Second),
		shutdownChannel: make(chan struct{}),
		packetChan:      make(chan types.EnhancedPacketInfo),
		blackChan:       make(chan types.BlackEvent),
		cachePacket:     make([]types.EnhancedPacketInfo, 0, configManager.Config.MaxPacketCount),
		countrySummary:  make(map[string]types.Summary),
		citySummary:     make(map[string]types.Summary),
		ethTypeSummary:  make(map[string]types.Summary),
		ipProtoSummary:  make(map[string]types.Summary),
		inputPackets:    make(map[string]types.InputPacket),
		daySummary:      make(map[string]types.Summary),
		matchSummary:    make(map[string]types.Summary),
		dstPortSummary:  make(map[uint16]types.Summary),
		matchPackets:    make(map[string][]types.EnhancedPacketInfo),
	}

	go sm.processPackets()
	return sm
}

func (sm *SummaryManager) Destroy() {
	if sm.summaryTicker != nil {
		sm.summaryTicker.Stop()
	}
	close(sm.packetChan)
	close(sm.blackChan)
	close(sm.shutdownChannel)

	if sm.geoipDB != nil {
		sm.geoipDB.Close()
	}
}

// 处理数据包
func (sm *SummaryManager) processPackets() {
	for {
		select {
		case <-sm.shutdownChannel:
			return
		case packet := <-sm.packetChan:
			sm.handlePacket(packet)
		case blackEvent := <-sm.blackChan:
			sm.handleBlackEvent(blackEvent)
		case <-sm.summaryTicker.C:
			sm.summarizeData()
		}
	}
}

// 处理数据包
func (sm *SummaryManager) handlePacket(packet types.EnhancedPacketInfo) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	if sm.geoipDB != nil && packet.SrcIP != "" {
		packet.Country = "-"
		packet.CountryCode = "-"
		packet.City = "-"
		if !utils.IsLocalIP(packet.SrcIP) {
			record, err := sm.geoipDB.City(net.ParseIP(packet.SrcIP))
			if err == nil {
				if record.Country.GeoNameID != 0 {
					packet.CountryCode = record.Country.IsoCode
					if country, ok := record.Country.Names["zh-CN"]; ok && country != "" {
						packet.Country = country
					} else {
						packet.Country = record.Country.Names["en"]
					}
					if city, ok := record.City.Names["zh-CN"]; ok && city != "" {
						packet.City = city
					} else {
						packet.City = record.City.Names["en"]
					}
				}
			}
		} else {
			packet.Country = "局域网"
			packet.City = "局域网"
		}
	}
	sm.cachePacket = append(sm.cachePacket, packet)
}

// 处理黑名单事件
func (sm *SummaryManager) handleBlackEvent(blackEvent types.BlackEvent) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	key := ""
	if blackEvent.IpVersion == 0 {
		mac := net.HardwareAddr(blackEvent.Mac[:]).String()
		if mac == "" {
			return
		}
		if _, ok := sm.blackSummary[mac]; !ok {
			sm.blackSummary[mac] = 0
		}
		key = mac
		sm.blackSummary[mac]++
	} else if blackEvent.IpVersion == 1 {
		ip := net.IP(blackEvent.Ip[:]).String()
		if ip == "" {
			return
		}
		if _, ok := sm.blackSummary[ip]; !ok {
			sm.blackSummary[ip] = 0
		}
		key = ip
		sm.blackSummary[ip]++
	} else if blackEvent.IpVersion == 2 {
		ipv6 := net.IP(blackEvent.Ip[:]).String()
		if ipv6 == "" {
			return
		}
		if _, ok := sm.blackSummary[ipv6]; !ok {
			sm.blackSummary[ipv6] = 0
		}
		key = ipv6
		sm.blackSummary[ipv6]++
	}
	if key != "" {
		sm.wsServer.BroadcastBlackEvent(key)
	}
}

// 汇总数据
func (sm *SummaryManager) summarizeData() {
	sm.mutex.Lock()
	packets := sm.cachePacket
	sm.cachePacket = make([]types.EnhancedPacketInfo, 0, len(packets))
	sm.mutex.Unlock()

	for _, packet := range packets {
		sm.updateSummary(sm.countrySummary, packet.Country, packet.PktSize)
		sm.updateSummary(sm.citySummary, packet.City, packet.PktSize)
		sm.updateSummary(sm.ethTypeSummary, packet.EthProto.String(), packet.PktSize)
		sm.updateSummary(sm.ipProtoSummary, packet.IPProto.String(), packet.PktSize)
		sm.updateSummary(sm.daySummary, time.Unix(0, packet.Timestamp).Format("2006-01-02"), packet.PktSize)

		if packet.DstPort != 0 {
			sm.updatePortSummary(packet.DstPort, packet.PktSize)
		}

		sm.updateInputPacket(packet)

		if sm.ruleMatcher != nil {
			sm.updateMatchSummary(packet)
		}
	}

	// 广播数据
	if len(sm.inputPackets) > 0 {
		sm.broadcastSummary()
	}
}

// 更新汇总数据
func (sm *SummaryManager) updateSummary(summaryMap map[string]types.Summary, key string, pktSize uint32) {
	if key == "" {
		return
	}
	summary := summaryMap[key]
	summary.Count++
	summary.Size += uint64(pktSize)
	summaryMap[key] = summary
}

// 更新目标端口数据包
func (sm *SummaryManager) updatePortSummary(port uint16, pktSize uint32) {
	summary := sm.dstPortSummary[port]
	summary.Count++
	summary.Size += uint64(pktSize)
	sm.dstPortSummary[port] = summary
}

// 更新输入数据包
func (sm *SummaryManager) updateInputPacket(packet types.EnhancedPacketInfo) {
	key := packet.SrcMAC + "-" + packet.SrcIP
	inputPacket, ok := sm.inputPackets[key]
	if !ok {
		inputPacket = types.InputPacket{
			SrcMAC:    packet.SrcMAC,
			SrcIP:     packet.SrcIP,
			Summary:   types.Summary{},
			Country:   packet.Country,
			City:      packet.City,
			StartTime: time.Now().Unix(),
			LastTime:  time.Now().Unix(),
			Target:    make(map[string]types.InputTarget),
		}
	}
	if packet.SrcIP != "" && packet.SrcIP != inputPacket.SrcIP {
		inputPacket.SrcIP = packet.SrcIP
	}
	if packet.Country != "" && packet.Country != inputPacket.Country {
		inputPacket.Country = packet.Country
	}
	if packet.City != "" && packet.City != inputPacket.City {
		inputPacket.City = packet.City
	}
	inputPacket.Summary.Count++
	inputPacket.Summary.Size += uint64(packet.PktSize)
	inputPacket.LastTime = time.Now().Unix()

	// 更新目标信息
	targetKey := packet.DstMAC + "-" + packet.DstIP
	target, ok := inputPacket.Target[targetKey]
	if !ok {
		target = types.InputTarget{
			Mac:       packet.DstMAC,
			Ip:        packet.DstIP,
			StartTime: time.Now().Unix(),
			LastTime:  time.Now().Unix(),
			Summary:   types.Summary{},
			Port:      make(map[uint16]types.Summary),
			EthType:   make(map[string]types.Summary),
			IPProto:   make(map[string]types.Summary),
		}
	}
	target.LastTime = time.Now().Unix()
	target.Summary.Count++
	target.Summary.Size += uint64(packet.PktSize)

	// 更新端口的统计信息
	portSummary := target.Port[packet.DstPort]
	portSummary.Count++
	portSummary.Size += uint64(packet.PktSize)
	target.Port[packet.DstPort] = portSummary

	updateSummary := func(m map[string]types.Summary, key string) {
		summary := m[key]
		summary.Count++
		summary.Size += uint64(packet.PktSize)
		m[key] = summary
	}
	// 更新以太网类型和IP协议的统计信息
	updateSummary(target.EthType, packet.EthProto.String())
	updateSummary(target.IPProto, packet.IPProto.String())

	inputPacket.Target[targetKey] = target
	sm.inputPackets[key] = inputPacket

	// 检查并限制 inputPackets 的大小
	if len(sm.inputPackets) > sm.configManager.Config.MaxPacketCount {
		var oldestKey string
		var oldestLastTime int64 = 0

		now := time.Now().Unix()
		for key, packet := range sm.inputPackets {
			// 计算最后访问时间与当前时间的差值
			timeDiff := now - packet.LastTime
			if timeDiff > oldestLastTime {
				oldestLastTime = timeDiff
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(sm.inputPackets, oldestKey)
		}
	}
}

// 更新匹配数据
func (sm *SummaryManager) updateMatchSummary(packet types.EnhancedPacketInfo) {
	matched, ruleName := sm.ruleMatcher.MatchPacket(packet)
	if !matched {
		return
	}
	summary := sm.matchSummary[ruleName]
	summary.Count++
	summary.Size += uint64(packet.PktSize)
	sm.matchSummary[ruleName] = summary

	if _, ok := sm.matchPackets[ruleName]; !ok {
		sm.matchPackets[ruleName] = make([]types.EnhancedPacketInfo, 0, sm.configManager.Config.MaxPacketCount)
	}
	sm.matchPackets[ruleName] = append(sm.matchPackets[ruleName], packet)
	if len(sm.matchPackets[ruleName]) > sm.configManager.Config.MaxPacketCount-1 {
		sm.matchPackets[ruleName] = sm.matchPackets[ruleName][len(sm.matchPackets[ruleName])-sm.configManager.Config.MaxPacketCount+1:]
	}
}

func (sm *SummaryManager) GetSummary() types.WebSocketSummaryPayload {
	return types.WebSocketSummaryPayload{
		CountrySummary: sm.countrySummary,
		CitySummary:    sm.citySummary,
		EthTypeSummary: sm.ethTypeSummary,
		IPProtoSummary: sm.ipProtoSummary,
		DaySummary:     sm.daySummary,
		MatchSummary:   sm.matchSummary,
		DstPortSummary: sm.dstPortSummary,
		InputPackets:   sm.inputPackets,
		BlackSummary:   sm.blackSummary,
	}
}

// 广播数据
func (sm *SummaryManager) broadcastSummary() {
	sm.wsServer.BroadcastSummary(sm.GetSummary())
}

func (sm *SummaryManager) GetMatchList(query *types.WebSocketMatchQueryPayload) []types.EnhancedPacketInfo {
	list := make([]types.EnhancedPacketInfo, 0)
	for ruleName, packets := range sm.matchPackets {
		if query.RuleName != "" && query.RuleName != ruleName {
			continue
		}
		for _, packet := range packets {
			if query.StartTime > 0 && packet.Timestamp < query.StartTime {
				continue
			}
			if query.EndTime > 0 && packet.Timestamp > query.EndTime {
				continue
			}
			if query.Country != "" && query.Country != packet.Country {
				continue
			}
			if query.City != "" && query.City != packet.City {
				continue
			}
			if query.SrcMAC != "" && query.SrcMAC != packet.SrcMAC {
				continue
			}
			if query.SrcIP != "" && query.SrcIP != packet.SrcIP {
				continue
			}
			if query.DstMAC != "" && query.DstMAC != packet.DstMAC {
				continue
			}
			if query.DstIP != "" && query.DstIP != packet.DstIP {
				continue
			}
			if query.EthType != 0 && query.EthType != uint16(packet.EthProto) {
				continue
			}
			if query.IPProto != 0 && query.IPProto != uint16(packet.IPProto) {
				continue
			}
			list = append(list, packet)
		}
	}
	if query.Order == "desc" {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Timestamp > list[j].Timestamp
		})
	} else {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Timestamp < list[j].Timestamp
		})
	}
	if query.Page > 0 && query.PageSize > 0 {
		start := (query.Page - 1) * query.PageSize
		end := start + query.PageSize
		list = list[start:end]
	}
	return list
}
