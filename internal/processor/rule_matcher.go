package processor

import (
	"net"
	"strconv"
	"strings"

	"github.com/danger-dream/ebpf-firewall/internal/enum"
	"github.com/danger-dream/ebpf-firewall/internal/types"
)

type MatchRuleIP struct {
	Type        int
	IP          string
	IPNet       *net.IPNet
	CountryCode string
	CityName    string
	Not         bool
}

func NewMatchRuleIP(ip string) *MatchRuleIP {
	localIP := strings.ToUpper(strings.TrimSpace(ip))
	// 国家匹配
	if strings.HasPrefix(localIP, "GEOIP:") {
		not := false
		if strings.HasPrefix(localIP, "GEOIP:!") {
			not = true
			localIP = strings.TrimPrefix(localIP, "GEOIP:!")
		} else {
			localIP = strings.TrimPrefix(localIP, "GEOIP:")
		}
		return &MatchRuleIP{Type: 3, CountryCode: localIP, Not: not}
	}
	// 城市匹配
	if strings.HasPrefix(localIP, "CITY:") {
		not := false
		if strings.HasPrefix(localIP, "CITY:!") {
			not = true
			localIP = strings.TrimPrefix(localIP, "CITY:!")
		} else {
			localIP = strings.TrimPrefix(localIP, "CITY:")
		}
		return &MatchRuleIP{Type: 4, CityName: localIP, Not: not}
	}
	// 网段匹配
	if strings.Contains(localIP, "/") {
		_, ipNet, err := net.ParseCIDR(localIP)
		if err != nil {
			return nil
		}
		return &MatchRuleIP{Type: 2, IPNet: ipNet}
	}
	// IP 匹配
	if net.ParseIP(localIP) != nil {
		return &MatchRuleIP{Type: 1, IP: localIP}
	}
	return nil
}

func (mri *MatchRuleIP) IsMatch(packet types.EnhancedPacketInfo) bool {
	if mri.Type == 1 {
		return mri.IP == packet.SrcIP || mri.IP == packet.DstIP
	} else if mri.Type == 2 {
		if packet.SrcIP != "" {
			return mri.IPNet.Contains(net.ParseIP(packet.SrcIP))
		}
		if packet.DstIP != "" {
			return mri.IPNet.Contains(net.ParseIP(packet.DstIP))
		}
	} else if mri.Type == 3 {
		// 国家匹配
		if packet.CountryCode != "" {
			match := mri.CountryCode == strings.ToUpper(packet.CountryCode)
			if mri.Not {
				return !match
			}
			return match
		}
	} else if mri.Type == 4 {
		// 城市匹配
		if packet.City != "" {
			match := mri.CityName == packet.City
			if mri.Not {
				return !match
			}
			return match
		}
	}
	return false
}

type MatchRulePort struct {
	Type      int
	Port      uint16
	StartPort uint16
	EndPort   uint16
	Ports     []uint16
}

func NewMatchRulePort(port interface{}) *MatchRulePort {
	switch port := port.(type) {
	case string:
		if strings.Contains(port, "-") {
			rangePorts := strings.Split(port, "-")
			start, err1 := strconv.Atoi(rangePorts[0])
			end, err2 := strconv.Atoi(rangePorts[1])
			if err1 != nil || err2 != nil {
				return nil
			}
			if start > 0 && start <= 65535 && end > 0 && end <= 65535 && start <= end {
				return &MatchRulePort{Type: 2, StartPort: uint16(start), EndPort: uint16(end)}
			}
		} else if strings.Contains(port, ",") {
			list := strings.Split(port, ",")
			ports := make([]uint16, 0, len(list))
			for _, p := range list {
				port, err := strconv.Atoi(p)
				if err != nil {
					continue
				}
				if port > 0 && port <= 65535 {
					ports = append(ports, uint16(port))
				}
			}
			if len(ports) > 0 {
				return &MatchRulePort{Type: 1, Ports: ports}
			}
		} else {
			port, err := strconv.Atoi(port)
			if err != nil {
				return nil
			}
			if port > 0 && port <= 65535 {
				return &MatchRulePort{Type: 0, Port: uint16(port)}
			}
		}
	case uint16:
	case uint32:
		if port > 0 && port <= 65535 {
			return &MatchRulePort{Type: 0, Port: uint16(port)}
		}
	}
	return nil
}

func (mrp *MatchRulePort) IsMatch(port uint16) bool {
	if mrp.Type == 0 {
		return mrp.Port == port
	} else if mrp.Type == 1 {
		for _, p := range mrp.Ports {
			if p == port {
				return true
			}
		}
	} else if mrp.Type == 2 {
		return port >= mrp.StartPort && port <= mrp.EndPort
	}
	return false
}

type MatchRuleMAC struct {
	MAC string
}

func NewMatchRuleMAC(mac string) *MatchRuleMAC {
	if mac, err := net.ParseMAC(mac); err == nil {
		return &MatchRuleMAC{MAC: strings.ToLower(strings.ReplaceAll(mac.String(), ":", ""))}
	}
	return nil
}

func (mm *MatchRuleMAC) IsMatchIgnoreFormat(mac string) bool {
	return mm.MAC == strings.ToLower(strings.ReplaceAll(mac, ":", ""))
}

func (mm *MatchRuleMAC) IsMatch(mac string) bool {
	return mm.MAC == mac
}

type MatchRuleEthType struct {
	EthType enum.EthernetType
}

func NewMatchRuleEthType(ethType string) *MatchRuleEthType {
	if et, ok := enum.EthernetTypeMap[strings.ToLower(ethType)]; ok {
		return &MatchRuleEthType{EthType: et}
	}
	return nil
}

func (met *MatchRuleEthType) IsMatch(ethType enum.EthernetType) bool {
	return met.EthType == ethType
}

type MatchRuleIPProtocol struct {
	IPProto enum.IPProtocol
}

func NewMatchRuleIPProtocol(ipProto string) *MatchRuleIPProtocol {
	if ip, ok := enum.IPProtocolMap[strings.ToLower(ipProto)]; ok {
		return &MatchRuleIPProtocol{IPProto: ip}
	}
	return nil
}

func (mip *MatchRuleIPProtocol) IsMatch(ipProto enum.IPProtocol) bool {
	return mip.IPProto == ipProto
}

type MatchRule struct {
	RuleName string
	IP       []*MatchRuleIP
	Port     []*MatchRulePort
	MAC      []*MatchRuleMAC
	EthType  []*MatchRuleEthType
	IPProto  []*MatchRuleIPProtocol
}

func (mr *MatchRule) IsMatch(packet types.EnhancedPacketInfo) bool {
	if len(mr.IP) > 0 {
		for _, ip := range mr.IP {
			if ip.IsMatch(packet) {
				return true
			}
		}
	}
	if len(mr.Port) > 0 {
		for _, port := range mr.Port {
			if port.IsMatch(packet.SrcPort) || port.IsMatch(packet.DstPort) {
				return true
			}
		}
	}
	if len(mr.MAC) > 0 {
		for _, mac := range mr.MAC {
			if mac.IsMatchIgnoreFormat(packet.SrcMAC) || mac.IsMatchIgnoreFormat(packet.DstMAC) {
				return true
			}
		}
	}
	if len(mr.EthType) > 0 {
		for _, ethType := range mr.EthType {
			if ethType.IsMatch(packet.EthProto) {
				return true
			}
		}
	}
	if len(mr.IPProto) > 0 {
		for _, ipProto := range mr.IPProto {
			if ipProto.IsMatch(packet.IPProto) {
				return true
			}
		}
	}
	return false
}

type RuleMatcher struct {
	rules    []MatchRule
	oldRules []types.Rule
}

func (rm *RuleMatcher) MatchPacket(packet types.EnhancedPacketInfo) (bool, string) {
	for _, rule := range rm.rules {
		if rule.IsMatch(packet) {
			return true, rule.RuleName
		}
	}
	return false, ""
}

func (rm *RuleMatcher) GetRules() []types.Rule {
	return rm.oldRules
}

func (rm *RuleMatcher) ParseRules(rules []types.Rule) int {
	list := make([]MatchRule, 0)
	if len(rules) > 0 {
		for _, rule := range rules {
			matchRule := MatchRule{
				RuleName: rule.RuleName,
			}
			ips := make([]*MatchRuleIP, 0)
			for _, ip := range rule.IP {
				mIP := NewMatchRuleIP(ip)
				if mIP != nil {
					ips = append(ips, mIP)
				}
			}
			if len(ips) > 0 {
				matchRule.IP = ips
			}
			ports := make([]*MatchRulePort, 0)
			for _, port := range rule.Port {
				mPort := NewMatchRulePort(port)
				if mPort != nil {
					ports = append(ports, mPort)
				}
			}
			if len(ports) > 0 {
				matchRule.Port = ports
			}
			macs := make([]*MatchRuleMAC, 0)
			for _, mac := range rule.MAC {
				mMAC := NewMatchRuleMAC(mac)
				if mMAC != nil {
					macs = append(macs, mMAC)
				}
			}
			if len(macs) > 0 {
				matchRule.MAC = macs
			}
			ethTypes := make([]*MatchRuleEthType, 0)
			for _, ethType := range rule.EthType {
				mEthType := NewMatchRuleEthType(ethType)
				if mEthType != nil {
					ethTypes = append(ethTypes, mEthType)
				}
			}
			if len(ethTypes) > 0 {
				matchRule.EthType = ethTypes
			}
			ipProtocols := make([]*MatchRuleIPProtocol, 0)
			for _, ipProto := range rule.IPProtocol {
				mIPProto := NewMatchRuleIPProtocol(ipProto)
				if mIPProto != nil {
					ipProtocols = append(ipProtocols, mIPProto)
				}
			}
			if len(ipProtocols) > 0 {
				matchRule.IPProto = ipProtocols
			}
			list = append(list, matchRule)
		}
	}
	rm.rules = list
	rm.oldRules = rules
	return len(list)
}

func NewRuleMatcher(rules []types.Rule) *RuleMatcher {
	rm := &RuleMatcher{}
	rm.ParseRules(rules)
	return rm
}
