package ebpf

import (
	"bytes"

	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/enum"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/types"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
)

type EBPFManager struct {
	configManager   *config.ConfigManager
	summaryManager  *SummaryManager
	interfaceName   string
	maxPacketCount  int
	ebpfObjects     *xdpObjects
	link            *link.Link
	perfReader      *perf.Reader
	blackPerfReader *perf.Reader
	isRunning       bool
	shutdownChannel chan struct{}
	cachePacket     []types.EnhancedPacketInfo
	linkType        string
}

func (em *EBPFManager) Start() error {
	if em.isRunning {
		return fmt.Errorf("ebpf 已启动")
	}
	iface, err := net.InterfaceByName(em.configManager.Config.Interface)
	if err != nil {
		return fmt.Errorf("获取接口 %s 失败: %s", em.configManager.Config.Interface, err)
	}
	em.maxPacketCount = em.configManager.Config.MaxPacketCount
	em.interfaceName = em.configManager.Config.Interface

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Printf("删除内存锁失败: %s\n", err.Error())
	}
	var ebpfObj xdpObjects
	if err := loadXdpObjects(&ebpfObj, nil); err != nil {
		return fmt.Errorf("加载 eBPF 对象失败: %s", err.Error())
	}
	em.ebpfObjects = &ebpfObj
	// attach nic offload mode: offload xdp
	linkPointer, linkErr := link.AttachXDP(link.XDPOptions{
		Program:   em.ebpfObjects.XdpProg,
		Interface: iface.Index,
		Flags:     link.XDPOffloadMode,
	})
	if linkErr != nil {
		// attach nic driver mode: native xdp
		linkPointer, linkErr = link.AttachXDP(link.XDPOptions{
			Program:   em.ebpfObjects.XdpProg,
			Interface: iface.Index,
			Flags:     link.XDPDriverMode,
		})
		if linkErr != nil {
			// attach nic generic mode: generic xdp
			linkPointer, linkErr = link.AttachXDP(link.XDPOptions{
				Program:   em.ebpfObjects.XdpProg,
				Interface: iface.Index,
				Flags:     link.XDPGenericMode,
			})
			if linkErr != nil {
				em.Shutdown()
				return fmt.Errorf("附加 XDP 程序失败: %s", linkErr.Error())
			} else {
				em.linkType = "generic"
				log.Printf("附加 XDP 程序成功，模式: generic\n")
			}
		} else {
			em.linkType = "driver"
			log.Printf("附加 XDP 程序成功，模式: driver\n")
		}
	} else {
		em.linkType = "offload"
		log.Printf("附加 XDP 程序成功，模式: offload\n")
	}
	em.link = &linkPointer
	em.perfReader, err = perf.NewReader(em.ebpfObjects.Events, os.Getpagesize())
	if err != nil {
		em.Shutdown()
		return fmt.Errorf("创建流量监控 perf 事件读取器失败: %s", err.Error())
	}
	em.blackPerfReader, err = perf.NewReader(em.ebpfObjects.BlackEvents, os.Getpagesize())
	if err != nil {
		em.Shutdown()
		return fmt.Errorf("创建黑名单 perf 事件读取器失败: %s", err.Error())
	}
	em.shutdownChannel = make(chan struct{})
	em.cachePacket = make([]types.EnhancedPacketInfo, 0, em.maxPacketCount)
	em.isRunning = true
	go em.monitorEvents()
	go em.monitorBlackList()
	go em.monitorBlackEvents()
	return nil
}

func (em *EBPFManager) GetLinkType() string {
	return em.linkType
}

// 监控黑名单变化
func (em *EBPFManager) monitorBlackList() {
	for {
		select {
		case <-em.shutdownChannel:
			return
		case black := <-em.configManager.GetBlackChannel():
			if black.Inc {
				if black.Type == "mac" {
					mac, _ := net.ParseMAC(black.Data)
					var macKey [6]byte
					copy(macKey[:], mac)
					em.ebpfObjects.MacBlacklist.Put(&macKey, uint8(1))
				} else if black.Type == "ipv4" {
					ip := net.ParseIP(black.Data).To4()
					ipv4Key := binary.BigEndian.Uint32(ip)
					em.ebpfObjects.Ipv4Blacklist.Put(&ipv4Key, uint8(1))
				} else if black.Type == "ipv6" {
					ip := net.ParseIP(black.Data).To16()
					var ipv6Key [4]uint32
					for i := 0; i < 4; i++ {
						ipv6Key[i] = binary.BigEndian.Uint32(ip[i*4 : (i+1)*4])
					}
					em.ebpfObjects.Ipv6Blacklist.Put(&ipv6Key, uint8(1))
				}
			} else {
				// 从黑名单中移除
				if black.Type == "mac" {
					mac, _ := net.ParseMAC(black.Data)
					var macKey [6]byte
					copy(macKey[:], mac)
					em.ebpfObjects.MacBlacklist.Delete(&macKey)
				} else if black.Type == "ipv4" {
					ip := net.ParseIP(black.Data).To4()
					ipv4Key := binary.BigEndian.Uint32(ip)
					em.ebpfObjects.Ipv4Blacklist.Delete(&ipv4Key)
				} else if black.Type == "ipv6" {
					ip := net.ParseIP(black.Data).To16()
					var ipv6Key [4]uint32
					for i := 0; i < 4; i++ {
						ipv6Key[i] = binary.BigEndian.Uint32(ip[i*4 : (i+1)*4])
					}
					em.ebpfObjects.Ipv6Blacklist.Delete(&ipv6Key)
				}
			}
		}
	}
}

func (em *EBPFManager) monitorEvents() {
	for {
		select {
		case <-em.shutdownChannel:
			return
		default:
			record, err := em.perfReader.Read()
			if err != nil {
				if err == perf.ErrClosed {
					em.Shutdown()
					err := em.Start()
					if err != nil {
						log.Fatalf("重新启动 eBPF 失败: %s", err.Error())
					}
					return
				}
				continue
			}
			var pi types.PacketInfo
			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &pi); err != nil {
				continue
			}
			var srcIP, dstIP string
			if pi.EthProto == enum.EthernetTypeIPv4 { // IPv4
				srcIP = net.IP(pi.SrcIP[:]).String()
				dstIP = net.IP(pi.DstIP[:]).String()
			} else if pi.EthProto == enum.EthernetTypeIPv6 { // IPv6
				srcIP = net.IP(pi.SrcIPv6[:]).String()
				dstIP = net.IP(pi.DstIPv6[:]).String()
			}
			enhancedInfo := types.EnhancedPacketInfo{
				SrcIP:     srcIP,
				DstIP:     dstIP,
				SrcPort:   binary.BigEndian.Uint16(pi.SrcPort[:]),
				DstPort:   binary.BigEndian.Uint16(pi.DstPort[:]),
				SrcMAC:    net.HardwareAddr(pi.SrcMAC[:]).String(),
				DstMAC:    net.HardwareAddr(pi.DstMAC[:]).String(),
				EthProto:  pi.EthProto,
				IPProto:   pi.IPProto,
				PktSize:   pi.PktSize,
				Timestamp: time.Now().UnixNano(),
			}
			em.summaryManager.packetChan <- enhancedInfo
		}
	}
}

// 监控黑名单事件
func (em *EBPFManager) monitorBlackEvents() {
	for {
		select {
		case <-em.shutdownChannel:
			return
		default:
			record, err := em.blackPerfReader.Read()
			if err != nil {
				continue
			}
			var blackEvent types.BlackEvent
			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &blackEvent); err != nil {
				continue
			}
			em.summaryManager.blackChan <- blackEvent
		}
	}
}

func (em *EBPFManager) Shutdown() {
	em.isRunning = false
	if em.shutdownChannel != nil {
		close(em.shutdownChannel)
	}
	if em.perfReader != nil {
		em.perfReader.Close()
	}
	if em.blackPerfReader != nil {
		em.blackPerfReader.Close()
	}
	if em.link != nil {
		(*em.link).Close()
	}
	if em.ebpfObjects != nil {
		em.ebpfObjects.Close()
	}
}

func NewEBPFManager(configManager *config.ConfigManager, summaryManager *SummaryManager) *EBPFManager {
	return &EBPFManager{
		configManager:  configManager,
		summaryManager: summaryManager,
	}
}
