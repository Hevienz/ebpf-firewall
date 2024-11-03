package processor

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/ebpf"
	"github.com/danger-dream/ebpf-firewall/internal/metrics"
	"github.com/danger-dream/ebpf-firewall/internal/threatintel"
	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
	"github.com/oschwald/geoip2-golang"
)

var (
	LocalNetworkLabel = "local"
	GeoIPURL          = "https://github.com/du5/geoip/raw/refs/heads/main/GeoLite2-City.tar.gz"
)

type WindowState struct {
	Count     int32
	FirstTime int64
}

type Processor struct {
	dataDir          string
	pool             *utils.ElasticPool[*types.PacketInfo]
	ebpfManager      *ebpf.EBPFManager
	collector        *metrics.MetricsCollector
	threatAggregator *threatintel.Aggregator
	geoipDB          *geoip2.Reader
	config           atomic.Value
	windowStates     sync.Map
	done             chan struct{}
}

func NewProcessor(pool *utils.ElasticPool[*types.PacketInfo], ebpfManager *ebpf.EBPFManager, collector *metrics.MetricsCollector) (*Processor, error) {
	systemConfig := config.GetConfig()
	dir := systemConfig.DataDir
	geoipDB := loadGeoip(filepath.Join(dir, systemConfig.GeoIPPath))

	// Initialize threat intelligence aggregator first to ensure default metadata is available
	// This allows the system to function properly even without explicit configuration
	threatAggregator, err := threatintel.NewAggregator(dir)
	if err != nil {
		return nil, err
	}
	p := &Processor{
		dataDir:          dir,
		pool:             pool,
		ebpfManager:      ebpfManager,
		collector:        collector,
		threatAggregator: threatAggregator,
		geoipDB:          geoipDB,
		windowStates:     sync.Map{},
		done:             make(chan struct{}),
	}
	if err := p.loadConfig(); err != nil {
		return nil, err
	}

	if err := p.threatAggregator.Initialize(p.getConfig().ThreatIntel.Feeds); err != nil {
		return nil, err
	}
	p.pool.SetProcessor(p.processPackets)
	go p.cleanupRoutine()
	return p, nil
}

func loadGeoip(path string) *geoip2.Reader {
	var geoipDB *geoip2.Reader
	if path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// download the latest GeoIP database
			if err := utils.DownloadGeoIPTarGZ(GeoIPURL, path); err != nil {
				log.Printf("failed to download GeoIP database: %v", err)
			}
		}
		if _, err := os.Stat(path); err == nil {
			// init GeoIP database
			geoipDB, err = geoip2.Open(path)
			if err != nil {
				log.Fatalf("failed to open GeoIP database: %v", err)
			} else {
				log.Printf("GeoIP database loaded: %s", path)
				return geoipDB
			}
		}
	} else {
		log.Println("GeoIP database path is not set, skip loading GeoIP database")
	}
	return nil
}

func (p *Processor) Close() error {
	close(p.done)
	p.saveConfig()
	p.threatAggregator.Close()
	if p.geoipDB != nil {
		p.geoipDB.Close()
	}
	return nil
}

func (p *Processor) processPackets(pi *types.PacketInfo) {
	packet := p.createPacket(pi)
	p.collector.CollectPacket(packet)

	if packet.MatchType != types.NoMatch {
		// drop the packet if it's not a normal packet
		return
	}

	if packet.SrcIP != "" {
		srcIP := packet.SrcIP
		config := p.getConfig()
		isLocalAndIgnored := config.ThreatIntel.IgnoreLocalNetwork && utils.IsLocalIP(srcIP)
		if !isLocalAndIgnored && p.threatAggregator.Contains(srcIP) {
			p.handleThreatIntelMatch(srcIP)
			return
		}
	}
}

func (p *Processor) createPacket(pi *types.PacketInfo) *types.Packet {
	var srcIP, dstIP string
	if pi.EthProto == types.EthernetType(0x0800) { // IPv4
		srcIP = net.IP(pi.SrcIP[:]).String()
		dstIP = net.IP(pi.DstIP[:]).String()
	} else if pi.EthProto == types.EthernetType(0x86DD) { // IPv6
		srcIP = net.IP(pi.SrcIPv6[:]).String()
		dstIP = net.IP(pi.DstIPv6[:]).String()
	}
	packet := &types.Packet{
		Timestamp: time.Now().Unix(),
		SrcMAC:    net.HardwareAddr(pi.SrcMAC[:]).String(),
		DstMAC:    net.HardwareAddr(pi.DstMAC[:]).String(),
		SrcIP:     srcIP,
		DstIP:     dstIP,
		SrcPort:   pi.SrcPort,
		DstPort:   pi.DstPort,
		Size:      pi.PktSize,
		EthType:   pi.EthProto,
		IPProto:   pi.IPProto,
		MatchType: pi.MatchType,
	}

	if packet.SrcIP != "" && p.geoipDB != nil {
		if utils.IsLocalIP(packet.SrcIP) {
			packet.Country = LocalNetworkLabel
			packet.City = LocalNetworkLabel
			return packet
		}
		record, err := p.geoipDB.City(net.ParseIP(packet.SrcIP))
		if err == nil && record.Country.GeoNameID != 0 {
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
	return packet
}

func (p *Processor) handleThreatIntelMatch(srcIP string) {
	config := p.getConfig()
	enable := config.ThreatIntel.MatchMode == MatchActionModeBlock
	if config.ThreatIntel.MatchMode == MatchActionModeThreshold {
		now := time.Now().Unix()
		window := config.ThreatIntel.MatchWindow
		threshold := config.ThreatIntel.MatchThreshold

		var state *WindowState
		if val, ok := p.windowStates.Load(srcIP); ok {
			state = val.(*WindowState)
			if now-state.FirstTime > int64(window) {
				state = &WindowState{
					Count:     1,
					FirstTime: now,
				}
				p.windowStates.Store(srcIP, state)
			} else {
				atomic.AddInt32(&state.Count, 1)
			}
		} else {
			state = &WindowState{
				Count:     1,
				FirstTime: now,
			}
			p.windowStates.Store(srcIP, state)
		}

		if state.Count >= int32(threshold) {
			enable = true
		} else {
			return
		}
	}
	expireTime := int64(0)
	if config.ThreatIntel.BlockDuration > 0 {
		expireTime = time.Now().Add(time.Duration(config.ThreatIntel.BlockDuration) * time.Second).Unix()
	}
	if err := p.AddBlockRule(&BlockRule{
		Value:      srcIP,
		Note:       "Matched threat intelligence",
		Source:     BlockSourceTypeIntel,
		CreateTime: time.Now().Unix(),
		Enabled:    enable,
		ExpireTime: expireTime,
	}); err != nil {
		log.Printf("Failed to add block rule for IP %s: %v", srcIP, err)
	}
}

func (p *Processor) updateBlockRuleToKernel(rule *BlockRule) error {
	if !rule.Enabled {
		return nil
	}

	if rule.ExpireTime > 0 && rule.ExpireTime <= time.Now().Unix() {
		rule.Enabled = false
		return nil
	}

	if err := p.ebpfManager.AddRule(rule.Value); err != nil {
		return err
	}
	return nil
}

func (p *Processor) cleanupWindowStates() {
	now := time.Now().Unix()
	window := p.getConfig().ThreatIntel.MatchWindow

	p.windowStates.Range(func(key, value interface{}) bool {
		state := value.(*WindowState)
		if now-state.FirstTime > int64(window) {
			p.windowStates.Delete(key)
		}
		return true
	})
}

func (p *Processor) cleanupBlockRules() {
	now := time.Now().Unix()

	p.updateConfig(func(config *ProcessorConfig) error {
		rules := config.Blocklist.Rules
		for i := len(rules) - 1; i >= 0; i-- {
			rule := &rules[i]
			if rule.ExpireTime > 0 && rule.ExpireTime <= now {
				if rule.Enabled {
					if err := p.ebpfManager.DeleteRule(rule.Value); err != nil {
						log.Printf("Failed to remove expired rule from kernel: %v", err)
						continue
					}
				}
				rules = append(rules[:i], rules[i+1:]...)
			}
		}
		config.Blocklist.Rules = rules
		return nil
	})
}

func (p *Processor) cleanupRoutine() {
	ticker := time.NewTicker(p.getConfig().CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.cleanupWindowStates()
			p.cleanupBlockRules()
		}
	}
}

func (p *Processor) GetBlockRules(page, pageSize int) ([]BlockRule, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	config := p.getConfig()
	total := len(config.Blocklist.Rules)
	start := (page - 1) * pageSize
	if start >= total {
		return []BlockRule{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return config.Blocklist.Rules[start:end], total, nil
}

func (p *Processor) AddBlockRule(rule *BlockRule) error {
	rule.ID = utils.GenerateUUID()
	if err := p.updateConfig(func(config *ProcessorConfig) error {
		if err := p.updateBlockRuleToKernel(rule); err != nil {
			return fmt.Errorf("failed to update block rule to kernel: %v", err)
		}
		config.Blocklist.Rules = append(config.Blocklist.Rules, *rule)
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (p *Processor) UpdateBlockRule(id string, rule BlockRule) error {
	return p.updateConfig(func(config *ProcessorConfig) error {
		for i := range config.Blocklist.Rules {
			if config.Blocklist.Rules[i].ID == id {
				oldEnabled := config.Blocklist.Rules[i].Enabled
				config.Blocklist.Rules[i] = rule
				if oldEnabled != rule.Enabled {
					if rule.Enabled {
						if err := p.updateBlockRuleToKernel(&rule); err != nil {
							return err
						}
					} else {
						if err := p.ebpfManager.DeleteRule(rule.Value); err != nil {
							return err
						}
					}
				}
				return nil
			}
		}
		return fmt.Errorf("rule not found: %s", id)
	})
}

func (p *Processor) DeleteBlockRule(id string) error {
	return p.updateConfig(func(config *ProcessorConfig) error {
		for i := range config.Blocklist.Rules {
			if config.Blocklist.Rules[i].ID == id {
				config.Blocklist.Rules = append(config.Blocklist.Rules[:i], config.Blocklist.Rules[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("rule not found: %s", id)
	})
}

func (p *Processor) GetThreatIntelAggregator() *threatintel.Aggregator {
	return p.threatAggregator
}
