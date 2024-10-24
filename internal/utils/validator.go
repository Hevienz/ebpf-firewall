package utils

import (
	"net"
	"strconv"
)

// CheckAddr check addr is valid or not, only support some common formats rather than
// use net.Dial to check due to domain is not allowed
func CheckAddr(addr string) bool {
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	// if port is not numeric, or not in the range of 0-65535, return false
	if p, err := strconv.Atoi(port); err != nil || p < 0 || p > 65535 {
		return false
	}
	if ip == "" {
		return true
	}
	return net.ParseIP(ip) != nil
}
