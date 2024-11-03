package iptrie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
)

type IPAddress []uint32

const BitsPerWord = 32

func NewIPAddress(ip net.IP) IPAddress {
	if ip == nil {
		return nil
	}
	coercedIP := ip.To4()
	parts := 1
	if coercedIP == nil {
		coercedIP = ip.To16()
		parts = 4
	}
	if coercedIP == nil {
		return nil
	}
	addr := make(IPAddress, parts)
	for i := 0; i < parts; i++ {
		idx := i * net.IPv4len
		addr[i] = binary.BigEndian.Uint32(coercedIP[idx : idx+net.IPv4len])
	}
	return addr
}

func (n IPAddress) Equal(n1 IPAddress) bool {
	if len(n) != len(n1) {
		return false
	}
	if len(n) == 1 {
		return n[0] == n1[0]
	}
	return n[0] == n1[0] && n[1] == n1[1] &&
		n[2] == n1[2] && n[3] == n1[3]
}

func (n IPAddress) Bit(position uint) (uint32, error) {
	wordIdx := len(n) - 1 - int(position>>5)
	if wordIdx < 0 || wordIdx >= len(n) {
		return 0, fmt.Errorf("bit position not valid")
	}
	return (n[wordIdx] >> (position & 31)) & 1, nil
}

func (n IPAddress) LeastCommonBitPosition(n1 IPAddress) (uint, error) {
	if len(n) != len(n1) {
		return 0, fmt.Errorf("network input version mismatch")
	}
	for i := 0; i < len(n); i++ {
		mask := uint32(1) << 31
		pos := uint(31)
		for ; mask > 0; mask >>= 1 {
			if n[i]&mask != n1[i]&mask {
				if i == 0 && pos == 31 {
					return 0, fmt.Errorf("no greatest common bit")
				}
				return (pos + 1) + uint(BitsPerWord)*uint(len(n)-i-1), nil
			}
			pos--
		}
	}
	return 0, nil
}

type IPNetwork struct {
	CIDR      *net.IPNet
	Address   IPAddress
	Netmask   IPAddress
	PrefixLen int
}

func NewIPNetwork(ipNet *net.IPNet) IPNetwork {
	ones, _ := ipNet.Mask.Size()
	return IPNetwork{
		CIDR:      ipNet,
		Address:   NewIPAddress(ipNet.IP),
		Netmask:   IPAddress(NewIPAddress(net.IP(ipNet.Mask))),
		PrefixLen: ones,
	}
}

func (n IPNetwork) Masked(ones int) IPNetwork {
	mask := net.CIDRMask(ones, len(n.Address)*BitsPerWord)

	return NewIPNetwork(&net.IPNet{
		IP:   n.CIDR.IP.Mask(mask),
		Mask: mask,
	})
}

func (n IPNetwork) Contains(nn IPAddress) bool {
	if len(n.Netmask) != len(nn) {
		return false
	}
	if (nn[0] & n.Netmask[0]) != n.Address[0] {
		return false
	}
	if len(nn) == 4 {
		return (nn[1]&n.Netmask[1]) == n.Address[1] &&
			(nn[2]&n.Netmask[2]) == n.Address[2] &&
			(nn[3]&n.Netmask[3]) == n.Address[3]
	}
	return true
}

func (n IPNetwork) LeastCommonBitPosition(n1 IPNetwork) (uint, error) {
	maskSize := n.PrefixLen
	if n1.PrefixLen < maskSize {
		maskSize = n1.PrefixLen
	}
	maskPosition := len(n1.Address)*BitsPerWord - maskSize
	lcb, err := n.Address.LeastCommonBitPosition(n1.Address)
	if err != nil {
		return 0, err
	}
	return uint(math.Max(float64(maskPosition), float64(lcb))), nil
}

func (n IPNetwork) Equal(n1 IPNetwork) bool {
	return n.CIDR.IP.Equal(n1.CIDR.IP) && bytes.Equal(n.CIDR.Mask, n1.CIDR.Mask)
}
