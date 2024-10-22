package enum

import (
	"fmt"
)

// EthernetType is an enumeration of ethernet type values, and acts as a decoder
// for any type it supports.
type EthernetType uint16

const (
	// EthernetTypeLLC is not an actual ethernet type.  It is instead a
	// placeholder we use in Ethernet frames that use the 802.3 standard of
	// srcmac|dstmac|length|LLC instead of srcmac|dstmac|ethertype.
	EthernetTypeLLC                         EthernetType = 0
	EthernetTypeLOOP                        EthernetType = 0x0060
	EthernetTypePUP                         EthernetType = 0x0200
	EthernetTypePUPAT                       EthernetType = 0x0201
	EthernetTypeIPv4                        EthernetType = 0x0800
	EthernetTypeX25                         EthernetType = 0x0805
	EthernetTypeARP                         EthernetType = 0x0806
	EthernetTypeRARP                        EthernetType = 0x8035
	EthernetTypeIPv6                        EthernetType = 0x86DD
	EthernetTypeCiscoDiscovery              EthernetType = 0x2000
	EthernetTypeNortelDiscovery             EthernetType = 0x01a2
	EthernetTypeTransparentEthernetBridging EthernetType = 0x6558
	EthernetTypeDot1Q                       EthernetType = 0x8100
	EthernetTypePPP                         EthernetType = 0x880b
	EthernetTypePPPoEDiscovery              EthernetType = 0x8863
	EthernetTypePPPoESession                EthernetType = 0x8864
	EthernetTypeMPLSUnicast                 EthernetType = 0x8847
	EthernetTypeMPLSMulticast               EthernetType = 0x8848
	EthernetTypeEAPOL                       EthernetType = 0x888e
	EthernetTypeERSPAN                      EthernetType = 0x88be
	EthernetTypeQinQ                        EthernetType = 0x88a8
	EthernetTypeLinkLayerDiscovery          EthernetType = 0x88cc
	EthernetTypeEthernetCTP                 EthernetType = 0x9000
	EthernetTypeBPQ                         EthernetType = 0x08FF
	EthernetTypeIEEEPUP                     EthernetType = 0x0a00
	EthernetTypeIEEEPUPAT                   EthernetType = 0x0a01
	EthernetTypeDEC                         EthernetType = 0x6000
	EthernetTypeDNADL                       EthernetType = 0x6001
	EthernetTypeDNARC                       EthernetType = 0x6002
	EthernetTypeDNART                       EthernetType = 0x6003
	EthernetTypeLAT                         EthernetType = 0x6004
	EthernetTypeDIAG                        EthernetType = 0x6005
	EthernetTypeCUST                        EthernetType = 0x6006
	EthernetTypeSCA                         EthernetType = 0x6007
	EthernetTypeATALK                       EthernetType = 0x809B
	EthernetTypeAARP                        EthernetType = 0x80F3
	EthernetTypeIPX                         EthernetType = 0x8137
	EthernetTypePAUSE                       EthernetType = 0x8808
	EthernetTypeSLOW                        EthernetType = 0x8809
	EthernetTypeWCCP                        EthernetType = 0x883E
	EthernetTypeATMMPOA                     EthernetType = 0x884c
	EthernetTypeATMFATE                     EthernetType = 0x8884
	EthernetTypeAOE                         EthernetType = 0x88A2
	EthernetTypeTIPC                        EthernetType = 0x88CA
	EthernetType1588                        EthernetType = 0x88F7
	EthernetTypeFCOE                        EthernetType = 0x8906
	EthernetTypeFIP                         EthernetType = 0x8914
	EthernetTypeEDSA                        EthernetType = 0xDADA
)

func (et EthernetType) String() string {
	switch et {
	case EthernetTypeLLC:
		return "LLC"
	case EthernetTypeIPv4:
		return "IPv4"
	case EthernetTypeARP:
		return "ARP"
	case EthernetTypeRARP:
		return "RARP"
	case EthernetTypeIPv6:
		return "IPv6"
	case EthernetTypeCiscoDiscovery:
		return "Cisco Discovery"
	case EthernetTypeNortelDiscovery:
		return "Nortel Discovery"
	case EthernetTypeTransparentEthernetBridging:
		return "Transparent Ethernet Bridging"
	case EthernetTypeDot1Q:
		return "802.1Q"
	case EthernetTypePPP:
		return "PPP"
	case EthernetTypePPPoEDiscovery:
		return "PPPoE Discovery"
	case EthernetTypePPPoESession:
		return "PPPoE Session"
	case EthernetTypeMPLSUnicast:
		return "MPLS Unicast"
	case EthernetTypeMPLSMulticast:
		return "MPLS Multicast"
	case EthernetTypeEAPOL:
		return "EAPOL"
	case EthernetTypeERSPAN:
		return "ERSPAN"
	case EthernetTypeQinQ:
		return "QinQ"
	case EthernetTypeLinkLayerDiscovery:
		return "Link Layer Discovery"
	case EthernetTypeEthernetCTP:
		return "Ethernet CTP"
	case EthernetTypeBPQ:
		return "BPQ"
	case EthernetTypeIEEEPUP:
		return "IEEE PUP"
	case EthernetTypeIEEEPUPAT:
		return "IEEE PUPAT"
	case EthernetTypeDEC:
		return "DEC"
	case EthernetTypeDNADL:
		return "DNA DL"
	case EthernetTypeDNARC:
		return "DNA RC"
	case EthernetTypeDNART:
		return "DNA RT"
	case EthernetTypeLAT:
		return "LAT"
	case EthernetTypeDIAG:
		return "DIAG"
	case EthernetTypeCUST:
		return "CUST"
	case EthernetTypeSCA:
		return "SCA"
	case EthernetTypeATALK:
		return "AppleTalk"
	case EthernetTypeAARP:
		return "AARP"
	case EthernetTypeIPX:
		return "IPX"
	case EthernetTypePAUSE:
		return "PAUSE"
	case EthernetTypeSLOW:
		return "SLOW"
	case EthernetTypeWCCP:
		return "WCCP"
	case EthernetTypeATMMPOA:
		return "ATM MPOA"
	case EthernetTypeATMFATE:
		return "ATM FATE"
	case EthernetTypeAOE:
		return "AoE"
	case EthernetTypeTIPC:
		return "TIPC"
	case EthernetType1588:
		return "IEEE 1588"
	case EthernetTypeFCOE:
		return "FCoE"
	case EthernetTypeFIP:
		return "FIP"
	case EthernetTypeEDSA:
		return "EDSA"
	default:
		return fmt.Sprintf("0x%04x", uint16(et))
	}
}

var EthernetTypeMap = map[string]EthernetType{
	"ip":                            EthernetTypeIPv4,
	"ipv4":                          EthernetTypeIPv4,
	"ipv6":                          EthernetTypeIPv6,
	"arp":                           EthernetTypeARP,
	"cisco discovery":               EthernetTypeCiscoDiscovery,
	"nortel discovery":              EthernetTypeNortelDiscovery,
	"transparent ethernet bridging": EthernetTypeTransparentEthernetBridging,
	"802.1q":                        EthernetTypeDot1Q,
	"ppp":                           EthernetTypePPP,
	"pppoe":                         EthernetTypePPP,
	"pppoe discovery":               EthernetTypePPPoEDiscovery,
	"pppoe session":                 EthernetTypePPPoESession,
	"mpls unicast":                  EthernetTypeMPLSUnicast,
	"mpls multicast":                EthernetTypeMPLSMulticast,
	"eapol":                         EthernetTypeEAPOL,
	"erspan":                        EthernetTypeERSPAN,
	"qinq":                          EthernetTypeQinQ,
	"link layer discovery":          EthernetTypeLinkLayerDiscovery,
	"ethernet ctp":                  EthernetTypeEthernetCTP,
	"bpq":                           EthernetTypeBPQ,
	"ieee pup":                      EthernetTypeIEEEPUP,
	"ieee pupat":                    EthernetTypeIEEEPUPAT,
	"dec":                           EthernetTypeDEC,
	"dna dl":                        EthernetTypeDNADL,
	"dna rc":                        EthernetTypeDNARC,
	"dna rt":                        EthernetTypeDNART,
	"lat":                           EthernetTypeLAT,
	"diag":                          EthernetTypeDIAG,
	"cust":                          EthernetTypeCUST,
	"sca":                           EthernetTypeSCA,
	"appletalk":                     EthernetTypeATALK,
	"aarp":                          EthernetTypeAARP,
	"ipx":                           EthernetTypeIPX,
	"pause":                         EthernetTypePAUSE,
	"slow":                          EthernetTypeSLOW,
	"wccp":                          EthernetTypeWCCP,
	"atm mpoa":                      EthernetTypeATMMPOA,
	"atm fate":                      EthernetTypeATMFATE,
	"aoe":                           EthernetTypeAOE,
	"tipc":                          EthernetTypeTIPC,
	"ieee 1588":                     EthernetType1588,
	"fcoe":                          EthernetTypeFCOE,
	"fip":                           EthernetTypeFIP,
	"edsa":                          EthernetTypeEDSA,
}

type IPProtocol uint16

const (
	IPProtocolIP              IPProtocol = 0
	IPProtocolICMPv4          IPProtocol = 1
	IPProtocolIGMP            IPProtocol = 2
	IPProtocolIPv4            IPProtocol = 4
	IPProtocolTCP             IPProtocol = 6
	IPProtocolUDP             IPProtocol = 17
	IPProtocolRUDP            IPProtocol = 27
	IPProtocolIPv6            IPProtocol = 41
	IPProtocolIPv6Routing     IPProtocol = 43
	IPProtocolIPv6Fragment    IPProtocol = 44
	IPProtocolGRE             IPProtocol = 47
	IPProtocolESP             IPProtocol = 50
	IPProtocolAH              IPProtocol = 51
	IPProtocolICMPv6          IPProtocol = 58
	IPProtocolNoNextHeader    IPProtocol = 59
	IPProtocolIPv6Destination IPProtocol = 60
	IPProtocolOSPF            IPProtocol = 89
	IPProtocolIPIP            IPProtocol = 94
	IPProtocolEtherIP         IPProtocol = 97
	IPProtocolVRRP            IPProtocol = 112
	IPProtocolSCTP            IPProtocol = 132
	IPProtocolUDPLite         IPProtocol = 136
	IPProtocolMPLSInIP        IPProtocol = 137
)

func (ip IPProtocol) String() string {
	switch ip {
	case IPProtocolIP:
		return "IP"
	case IPProtocolICMPv4:
		return "ICMPv4"
	case IPProtocolTCP:
		return "TCP"
	case IPProtocolUDP:
		return "UDP"
	case IPProtocolIGMP:
		return "IGMP"
	case IPProtocolRUDP:
		return "RUDP"
	case IPProtocolIPv4:
		return "IPv4"
	case IPProtocolIPv6:
		return "IPv6"
	case IPProtocolICMPv6:
		return "ICMPv6"
	case IPProtocolIPv6Routing:
		return "IPv6 Routing"
	case IPProtocolIPv6Fragment:
		return "IPv6 Fragment"
	case IPProtocolGRE:
		return "GRE"
	case IPProtocolESP:
		return "ESP"
	case IPProtocolAH:
		return "AH"
	case IPProtocolOSPF:
		return "OSPF"
	case IPProtocolIPIP:
		return "IPIP"
	case IPProtocolEtherIP:
		return "EtherIP"
	case IPProtocolVRRP:
		return "VRRP"
	case IPProtocolSCTP:
		return "SCTP"
	case IPProtocolUDPLite:
		return "UDPLite"
	case IPProtocolMPLSInIP:
		return "MPLS-in-IP"
	case IPProtocolNoNextHeader:
		return "No Next Header"
	case IPProtocolIPv6Destination:
		return "IPv6 Destination"
	default:
		return fmt.Sprintf("%d", uint8(ip))
	}
}

var IPProtocolMap = map[string]IPProtocol{
	"icmp":             IPProtocolICMPv4,
	"icmpv4":           IPProtocolICMPv4,
	"igmp":             IPProtocolIGMP,
	"udp":              IPProtocolUDP,
	"tcp":              IPProtocolTCP,
	"rudp":             IPProtocolRUDP,
	"ip":               IPProtocolIPv4,
	"ipv4":             IPProtocolIPv4,
	"ipv6":             IPProtocolIPv6,
	"icmpv6":           IPProtocolICMPv6,
	"ipv6 routing":     IPProtocolIPv6Routing,
	"ipv6 fragment":    IPProtocolIPv6Fragment,
	"gre":              IPProtocolGRE,
	"esp":              IPProtocolESP,
	"ah":               IPProtocolAH,
	"ospf":             IPProtocolOSPF,
	"ipip":             IPProtocolIPIP,
	"etherip":          IPProtocolEtherIP,
	"vrrp":             IPProtocolVRRP,
	"sctp":             IPProtocolSCTP,
	"udplite":          IPProtocolUDPLite,
	"mpls-in-ip":       IPProtocolMPLSInIP,
	"no next header":   IPProtocolNoNextHeader,
	"ipv6 destination": IPProtocolIPv6Destination,
}
