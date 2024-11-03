package middleware

import "time"

// IPRecord holds the count and last reset time for an IP address
type IPRecord struct {
	// the number of requests for the IP address
	Count int
	// the timestamp of the last reset
	LastReset int64
}

// rate limiter for IP addresses
type Limiter struct {
	// maximum number of requests allowed per interval
	rateLimitRequest int
	// time interval in seconds for rate limiting
	rateLimitInterval int
	// map of IP addresses to their request counts and last reset times
	ipMap map[string]*IPRecord
	// channel to signal the cleanup goroutine to stop
	channel chan struct{}
}

func NewLimiter(rateLimitRequest int, rateLimitInterval int) *Limiter {
	if rateLimitRequest <= 0 {
		rateLimitRequest = 120
	}
	if rateLimitInterval <= 0 {
		rateLimitInterval = 60
	}
	limiter := Limiter{
		rateLimitRequest:  rateLimitRequest,
		rateLimitInterval: rateLimitInterval,
		ipMap:             make(map[string]*IPRecord, 1024),
		channel:           make(chan struct{}),
	}
	go limiter.cleanup()
	return &limiter
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(time.Second * time.Duration(l.rateLimitInterval))
	defer ticker.Stop()
	for {
		select {
		case <-l.channel:
			return
		case <-ticker.C:
			now := time.Now().Unix()
			for ip, record := range l.ipMap {
				if now-record.LastReset >= int64(l.rateLimitInterval) {
					delete(l.ipMap, ip)
				}
			}
		}
	}
}

// check if an IP address is rate limited
func (l *Limiter) IsRateLimited(ip string) bool {
	now := time.Now().Unix()
	if _, ok := l.ipMap[ip]; !ok {
		l.ipMap[ip] = &IPRecord{
			Count:     1,
			LastReset: now,
		}
		// not rate limited
		return false
	}
	record := l.ipMap[ip]
	// if the last reset time is greater than the interval, reset the count and last reset time
	if now-record.LastReset > int64(l.rateLimitInterval) {
		record.Count = 1
		record.LastReset = now
		return false
	}
	// increment the count for the IP address
	record.Count++
	// check if the IP address is rate limited
	return record.Count > l.rateLimitRequest
}

func (l *Limiter) Close() {
	close(l.channel)
}
