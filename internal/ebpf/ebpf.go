package ebpf

import (
	"bytes"
	"errors"
	"strings"

	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
)

type EBPFManager struct {
	interfaceName string
	objects       *xdpObjects
	link          *link.Link
	reader        *perf.Reader
	pool          *utils.ElasticPool[*types.PacketInfo]
	done          chan struct{}
	linkType      string
}

func NewEBPFManager(pool *utils.ElasticPool[*types.PacketInfo]) *EBPFManager {
	return &EBPFManager{pool: pool}
}

func (em *EBPFManager) Start() error {
	config := config.GetConfig()
	iface, err := net.InterfaceByName(config.Interface)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %s", config.Interface, err)
	}
	em.interfaceName = config.Interface

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Printf("failed to remove memlock: %s", err.Error())
	}
	var ebpfObj xdpObjects
	if err := loadXdpObjects(&ebpfObj, nil); err != nil {
		return fmt.Errorf("failed to load eBPF objects: %s", err.Error())
	}
	em.objects = &ebpfObj
	err = em.attachXDP(iface.Index)
	if err != nil {
		em.Close()
		return err
	}
	em.reader, err = perf.NewReader(em.objects.Events, os.Getpagesize())
	if err != nil {
		em.Close()
		return fmt.Errorf("failed to create perf event reader: %s", err.Error())
	}
	em.done = make(chan struct{})
	em.pool.SetProducer(em.monitorEvents)
	return nil
}

func (em *EBPFManager) attachXDP(index int) error {
	flagNames := []string{"offload", "driver", "generic"}
	errs := []string{}
	for i, mode := range []link.XDPAttachFlags{link.XDPOffloadMode, link.XDPDriverMode, link.XDPGenericMode} {
		flagName := flagNames[i]
		l, err := link.AttachXDP(link.XDPOptions{
			Program:   em.objects.XdpProg,
			Interface: index,
			Flags:     mode,
		})
		if err == nil {
			em.linkType = flagName
			em.link = &l
			log.Printf("XDP program attached successfully, current mode: %s", flagName)
			return nil
		}
		errs = append(errs, fmt.Sprintf("failed to attach XDP program with %s mode: %s", flagName, err.Error()))
	}
	return errors.New(strings.Join(errs, "\n"))
}

func (em *EBPFManager) monitorEvents(submit func(*types.PacketInfo)) {
	for {
		select {
		case <-em.done:
			return
		default:
			record, err := em.reader.Read()
			if err != nil {
				if err == perf.ErrClosed {
					log.Printf("perf event reader closed, trying to restart eBPF")
					em.Close()
					if err := em.Start(); err != nil {
						log.Fatalf("failed to restart eBPF: %s", err.Error())
					} else {
						log.Printf("eBPF restarted successfully")
					}
					return
				}
				continue
			}
			var pi types.PacketInfo
			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &pi); err != nil {
				continue
			}
			submit(&pi)
		}
	}
}

func (em *EBPFManager) Close() error {
	if em.done != nil {
		close(em.done)
	}
	if em.reader != nil {
		em.reader.Close()
	}
	if em.link != nil {
		(*em.link).Close()
	}
	if em.objects != nil {
		em.objects.Close()
	}
	return nil
}

func (em *EBPFManager) GetLinkType() string {
	return em.linkType
}

func (em *EBPFManager) updateMap(iptype utils.IPType, value []byte, add bool) (err error) {
	switch iptype {
	case utils.IPTypeIPv4:
		if add {
			err = em.objects.Ipv4List.Put(value, 1)
		} else {
			err = em.objects.Ipv4List.Delete(value)
		}
	case utils.IPTypeIPV4CIDR:
		if add {
			err = em.objects.Ipv4CidrTrie.Put(value, 1)
		} else {
			err = em.objects.Ipv4CidrTrie.Delete(value)
		}
	case utils.IPTypeIPv6:
		if add {
			err = em.objects.Ipv6List.Put(value, 1)
		} else {
			err = em.objects.Ipv6List.Delete(value)
		}
	case utils.IPTypeIPv6CIDR:
		if add {
			err = em.objects.Ipv6CidrTrie.Put(value, 1)
		} else {
			err = em.objects.Ipv6CidrTrie.Delete(value)
		}
	case utils.IPTypeMAC:
		if add {
			err = em.objects.MacList.Put(value, 1)
		} else {
			err = em.objects.MacList.Delete(value)
		}
	default:
		return fmt.Errorf("unsupported match type: %v", iptype)
	}
	return err
}

func (em *EBPFManager) AddRule(value string) error {
	bytes, iptype, err := utils.ParseValueToBytes(value)
	if err != nil {
		return err
	}
	return em.updateMap(iptype, bytes, true)
}

func (em *EBPFManager) DeleteRule(value string) error {
	bytes, iptype, err := utils.ParseValueToBytes(value)
	if err != nil {
		return err
	}
	return em.updateMap(iptype, bytes, false)
}
