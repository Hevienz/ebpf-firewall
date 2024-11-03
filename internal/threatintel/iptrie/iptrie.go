package iptrie

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

type IPTrie struct {
	mu       sync.RWMutex
	ipv4Root *prefixNode
	ipv6Root *prefixNode
}

func NewIPTrie() *IPTrie {
	_, rootNet, _ := net.ParseCIDR("0.0.0.0/0")
	_, rootNet6, _ := net.ParseCIDR("0::0/0")
	return &IPTrie{
		ipv4Root: &prefixNode{
			children: make([]*prefixNode, 2, 8),
			skipBits: 0,
			network:  NewIPNetwork(rootNet),
		},
		ipv6Root: &prefixNode{
			children: make([]*prefixNode, 2, 8),
			skipBits: 0,
			network:  NewIPNetwork(rootNet6),
		},
	}
}

func (t *IPTrie) Insert(addr string) error {
	ipNet := parseIPAddrToIPNet(addr)
	if ipNet == nil {
		return fmt.Errorf("invalid IP or CIDR: %s", addr)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if ipNet.IP.To4() != nil {
		return t.ipv4Root.Insert(ipNet)
	}
	return t.ipv6Root.Insert(ipNet)
}

func (t *IPTrie) Contains(addr string) bool {
	ipNet := parseIPAddrToIPNet(addr)
	if ipNet == nil {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if ipNet.IP.To4() != nil {
		return t.ipv4Root.Contains(ipNet.IP)
	}
	return t.ipv6Root.Contains(ipNet.IP)
}

func (t *IPTrie) Remove(addr string) error {
	ipNet := parseIPAddrToIPNet(addr)
	if ipNet == nil {
		return fmt.Errorf("invalid IP or CIDR: %s", addr)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if ipNet.IP.To4() != nil {
		return t.ipv4Root.Remove(ipNet)
	}
	return t.ipv6Root.Remove(ipNet)
}

func (t *IPTrie) Size() int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ipv4Root.Size() + t.ipv6Root.Size()
}

func (t *IPTrie) String() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ipv4Root.String() + "\n" + t.ipv6Root.String()
}

func parseIPAddrToIPNet(addr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(addr)
	if err != nil {
		ip := net.ParseIP(addr)
		if ip == nil {
			return nil
		}
		if ip.To4() != nil {
			ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
		} else {
			ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
		}
	}
	return ipNet
}

type prefixNode struct {
	parent    *prefixNode
	children  []*prefixNode
	skipBits  uint
	network   IPNetwork
	isLeaf    bool
	nodeCount int32
}

func newPathNode(network IPNetwork, numBitsSkipped uint) *prefixNode {
	path := &prefixNode{
		children: make([]*prefixNode, 2),
		skipBits: numBitsSkipped,
		network:  network.Masked(int(numBitsSkipped)),
	}
	return path
}

func newLeafNode(network IPNetwork) *prefixNode {
	leaf := newPathNode(network, uint(network.PrefixLen))
	leaf.isLeaf = true
	return leaf
}

func (p *prefixNode) Insert(entry *net.IPNet) error {
	n := NewIPNetwork(entry)
	sizeIncreased, err := p.insertNetwork(n)
	if sizeIncreased {
		atomic.AddInt32(&p.nodeCount, 1)
	} else if err == nil {
		return fmt.Errorf("network already exists")
	}
	return err
}

func (p *prefixNode) Remove(network *net.IPNet) error {
	err := p.removeNetwork(NewIPNetwork(network))
	if err != nil {
		return err
	}
	atomic.AddInt32(&p.nodeCount, -1)
	return nil
}

func (p *prefixNode) Contains(ip net.IP) bool {
	ipa := NewIPAddress(ip)
	if ipa == nil {
		return false
	}
	return p.containsAddress(ipa)
}

func (p *prefixNode) Size() int32 {
	return p.nodeCount
}

func (p *prefixNode) String() string {
	children := []string{}
	level := 0
	for parent := p.parent; parent != nil; parent = parent.parent {
		level++
	}
	padding := strings.Repeat("| ", level+1)
	for bits, child := range p.children {
		if child == nil {
			continue
		}
		childStr := fmt.Sprintf("\n%s%d--> %s", padding, bits, child.String())
		children = append(children, childStr)
	}
	return fmt.Sprintf("%s (target pos:%d, is_leaf:%t, mask:%d)%s", p.network.CIDR.IP.String(),
		p.getTargetBitPos(), p.isLeaf, p.network.PrefixLen, strings.Join(children, ""))
}

func (p *prefixNode) containsAddress(number IPAddress) bool {
	if !p.network.Contains(number) {
		return false
	}
	if p.isLeaf {
		return true
	}
	targetPos := p.getTargetBitPos()
	if targetPos < 0 {
		return false
	}
	bit, err := number.Bit(uint(targetPos))
	if err != nil {
		return false
	}
	child := p.children[bit]
	if child == nil {
		return false
	}
	return child.containsAddress(number)
}

func (p *prefixNode) insertNetwork(network IPNetwork) (bool, error) {
	current := p

	for {
		if current.network.Equal(network) {
			if current.isLeaf {
				return false, nil
			}
			current.isLeaf = true
			return true, nil
		}

		bit, err := current.getBitFromAddress(network.Address)
		if err != nil {
			return false, err
		}

		child := current.children[bit]
		if child == nil {
			current.children[bit] = newLeafNode(network)
			current.children[bit].parent = current
			return true, nil
		}

		lcb, err := network.LeastCommonBitPosition(child.network)
		if err != nil {
			return false, err
		}
		divergingBitPos := int(lcb) - 1
		if divergingBitPos <= child.getTargetBitPos() {
			current = child
			continue
		}
		pathNode := newPathNode(network, current.getTotalBits()-lcb)
		if err := current.insertPrefix(bit, pathNode, child); err != nil {
			return false, err
		}
		current = pathNode
	}
}

func (p *prefixNode) insertPrefix(bit uint32, pathPrefix, child *prefixNode) error {
	p.children[bit] = pathPrefix
	pathPrefix.parent = p

	pathPrefixBit, err := pathPrefix.getBitFromAddress(child.network.Address)
	if err != nil {
		return err
	}
	pathPrefix.children[pathPrefixBit] = child
	child.parent = pathPrefix
	return nil
}

func (p *prefixNode) removeNetwork(network IPNetwork) error {
	if p.isLeaf && p.network.Equal(network) {
		p.isLeaf = false
		return p.compressPath()
	}

	if p.getTargetBitPos() < 0 {
		return nil
	}

	bit, err := p.getBitFromAddress(network.Address)
	if err != nil {
		return err
	}
	child := p.children[bit]
	if child != nil {
		return child.removeNetwork(network)
	}
	return nil
}

func (p *prefixNode) canCompressPath() bool {
	return !p.isLeaf && p.getChildCount() <= 1 && p.parent != nil
}

func (p *prefixNode) compressPath() error {
	if !p.canCompressPath() {
		return nil
	}

	var loneChild *prefixNode
	for _, child := range p.children {
		if child != nil {
			loneChild = child
			break
		}
	}
	if loneChild == nil {
		if p.parent != nil {
			for i, child := range p.parent.children {
				if child == p {
					p.parent.children[i] = nil
					break
				}
			}
		}
		return nil
	}

	parent := p.parent
	for ; parent.canCompressPath(); parent = parent.parent {
	}

	parentBit, err := parent.getBitFromAddress(p.network.Address)
	if err != nil {
		return err
	}

	parent.children[parentBit] = loneChild
	loneChild.parent = parent

	return parent.compressPath()
}

func (p *prefixNode) getChildCount() int {
	count := 0
	for _, child := range p.children {
		if child != nil {
			count++
		}
	}
	return count
}

func (p *prefixNode) getTotalBits() uint {
	return BitsPerWord * uint(len(p.network.Address))
}

func (p *prefixNode) getTargetBitPos() int {
	return int(p.getTotalBits()) - int(p.skipBits) - 1
}

func (p *prefixNode) getBitFromAddress(n IPAddress) (uint32, error) {
	return n.Bit(uint(p.getTargetBitPos()))
}
