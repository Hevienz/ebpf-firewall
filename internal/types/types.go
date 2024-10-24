package types

import (
	"github.com/danger-dream/ebpf-firewall/internal/enum"
)

type Rule struct {
	RuleName   string        `json:"rule_name" yaml:"rule_name"`
	IP         []string      `json:"ip" yaml:"ip"`
	Port       []interface{} `json:"port" yaml:"port"`
	MAC        []string      `json:"mac" yaml:"mac"`
	EthType    []string      `json:"eth_type" yaml:"eth_type"`
	IPProtocol []string      `json:"ip_protocol" yaml:"ip_protocol"`
}

type ConfigBlack struct {
	Mac  []string `json:"mac" yaml:"mac"`
	Ipv4 []string `json:"ipv4" yaml:"ipv4"`
	Ipv6 []string `json:"ipv6" yaml:"ipv6"`
}

type Config struct {
	Interface      string      `json:"interface" yaml:"interface"`
	SummaryTime    int         `json:"summary_time" yaml:"summary_time"`
	Addr           string      `json:"addr" yaml:"addr"`
	Enable         bool        `json:"enable" yaml:"enable"`
	Rules          []Rule      `json:"rule" yaml:"rule"`
	MaxPacketCount int         `json:"max_packet_count" yaml:"max_packet_count"`
	GeoIPPath      string      `json:"geoip_path" yaml:"geoip_path"`
	Black          ConfigBlack `json:"black" yaml:"black"`
}

type BlackEvent struct {
	Mac       [6]byte
	Ip        [16]byte // 大小足够容纳 IPv6 地址
	IpVersion uint16
}

type PacketInfo struct {
	SrcIP    [4]byte
	DstIP    [4]byte
	SrcIPv6  [16]byte
	DstIPv6  [16]byte
	SrcPort  [2]byte
	DstPort  [2]byte
	SrcMAC   [6]byte
	DstMAC   [6]byte
	EthProto enum.EthernetType
	IPProto  enum.IPProtocol
	PktSize  uint32
}

type EnhancedPacketInfo struct {
	SrcIP       string            `json:"src_ip"`
	DstIP       string            `json:"dst_ip"`
	SrcPort     uint16            `json:"src_port"`
	DstPort     uint16            `json:"dst_port"`
	SrcMAC      string            `json:"src_mac"`
	DstMAC      string            `json:"dst_mac"`
	EthProto    enum.EthernetType `json:"eth_proto"`
	IPProto     enum.IPProtocol   `json:"ip_proto"`
	PktSize     uint32            `json:"pkt_size"`
	Timestamp   int64             `json:"timestamp"`
	Country     string            `json:"country"`
	CountryCode string            `json:"country_code"`
	City        string            `json:"city"`
}

type InputTarget struct {
	Mac       string             `json:"mac"`
	Ip        string             `json:"ip"`
	StartTime int64              `json:"start_time"`
	LastTime  int64              `json:"last_time"`
	Summary   Summary            `json:"summary"`
	Port      map[uint16]Summary `json:"port"`
	EthType   map[string]Summary `json:"eth_type"`
	IPProto   map[string]Summary `json:"ip_proto"`
}

type InputPacket struct {
	SrcMAC    string                 `json:"src_mac"`
	SrcIP     string                 `json:"src_ip"`
	Summary   Summary                `json:"summary"`
	Country   string                 `json:"country"`
	City      string                 `json:"city"`
	StartTime int64                  `json:"start_time"`
	LastTime  int64                  `json:"last_time"`
	Target    map[string]InputTarget `json:"target"`
}

type Summary struct {
	Count uint64 `json:"count"`
	Size  uint64 `json:"size"`
}

type WebSocketMatchQueryPayload struct {
	ID        string `json:"id"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	RuleName  string `json:"rule_name"`
	Order     string `json:"order"`
	Country   string `json:"country"`
	City      string `json:"city"`
	SrcMAC    string `json:"src_mac"`
	SrcIP     string `json:"src_ip"`
	DstMAC    string `json:"dst_mac"`
	DstIP     string `json:"dst_ip"`
	EthType   uint16 `json:"eth_type"`
	IPProto   uint16 `json:"ip_proto"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

type WebSocketChangeBlackListPayload struct {
	Inc  bool   `json:"inc"`
	Type string `json:"type"`
	Data string `json:"data"`
}

type WebSocketSummaryPayload struct {
	CountrySummary map[string]Summary     `json:"country_summary"`
	CitySummary    map[string]Summary     `json:"city_summary"`
	EthTypeSummary map[string]Summary     `json:"eth_type_summary"`
	IPProtoSummary map[string]Summary     `json:"ip_proto_summary"`
	DaySummary     map[string]Summary     `json:"day_summary"`
	MatchSummary   map[string]Summary     `json:"match_summary"`
	DstPortSummary map[uint16]Summary     `json:"dst_port_summary"`
	InputPackets   map[string]InputPacket `json:"input_packets"`
	BlackSummary   map[string]uint64      `json:"black_summary"`
}
