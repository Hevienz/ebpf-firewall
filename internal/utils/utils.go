package utils

import (
	"io"
	"log"
	"net"
)

var (
	localIPs []string = []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"233.252.0.0/24",
		"240.0.0.0/4",
		"255.255.255.255/32",
	}
	localIPNets []*net.IPNet
)

func init() {
	for _, ip := range localIPs {
		_, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			continue
		}
		localIPNets = append(localIPNets, ipNet)
	}
}

func IsLocalIP(ip string) bool {
	ipNet := net.ParseIP(ip)
	for _, localIPNet := range localIPNets {
		if localIPNet.Contains(ipNet) {
			return true
		}
	}
	return false
}

func GetDefaultInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Failed to get network interfaces:", err)
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return iface.Name
					}
				}
			}
		}
	}
	log.Fatal("No suitable network interface found")
	return ""
}

func SafeClose(c io.Closer, name string) {
	if err := c.Close(); err != nil {
		log.Printf("Error closing %s: %v", name, err)
	}
}

func RemoveStringFromSlice(slice []string, str string) []string {
	for i, s := range slice {
		if s == str {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func IsValidMAC(mac string) bool {
	_, err := net.ParseMAC(mac)
	return err == nil
}

func IsValidIPv4(ip string) bool {
	return net.ParseIP(ip) != nil && net.ParseIP(ip).To4() != nil
}

func IsValidIPv6(ip string) bool {
	return net.ParseIP(ip) != nil && net.ParseIP(ip).To4() == nil
}
