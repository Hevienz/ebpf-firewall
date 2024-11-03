package middleware

import (
	"fmt"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	// Initialize rate limiter: 3 requests per 5 seconds
	limiter := NewLimiter(3, 5)
	defer limiter.Close()
	testIP := "192.168.1.1"

	// Test Case 1: First access
	if limiter.IsRateLimited(testIP) {
		t.Error("Expected first access to not be rate limited")
	}

	// Test Case 2: Normal access within limits
	for i := 0; i < 2; i++ {
		if limiter.IsRateLimited(testIP) {
			t.Errorf("Expected access %d to not be rate limited", i+2)
		}
	}

	// Test Case 3: Exceeding limit
	if !limiter.IsRateLimited(testIP) {
		t.Error("Expected fourth access to be rate limited")
	}

	// Test Case 4: After reset period
	record := limiter.ipMap[testIP]
	record.LastReset -= 6 // Simulate 6 seconds passing

	if limiter.IsRateLimited(testIP) {
		t.Error("Expected access after reset to not be rate limited")
	}
}

func TestLimiterConcurrent(t *testing.T) {
	limiter := NewLimiter(100, 2)
	defer limiter.Close()
	testIP := "192.168.1.2"

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			limiter.IsRateLimited(testIP)
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	if limiter.ipMap[testIP].Count != 100 {
		t.Errorf("Expected request count to be 100, got %d", limiter.ipMap[testIP].Count)
	}
}

func TestMultipleIPs(t *testing.T) {
	limiter := NewLimiter(2, 5)
	defer limiter.Close()
	ip1 := "192.168.1.3"
	ip2 := "192.168.1.4"

	// Test IP1 rate limiting
	if limiter.IsRateLimited(ip1) {
		t.Error("Expected IP1 first access to not be rate limited")
	}
	if limiter.IsRateLimited(ip1) {
		t.Error("Expected IP1 second access to not be rate limited")
	}
	if !limiter.IsRateLimited(ip1) {
		t.Error("Expected IP1 third access to be rate limited")
	}

	// Test IP2 rate limiting (should not be affected by IP1)
	if limiter.IsRateLimited(ip2) {
		t.Error("Expected IP2 first access to not be rate limited")
	}
	if limiter.IsRateLimited(ip2) {
		t.Error("Expected IP2 second access to not be rate limited")
	}
	if !limiter.IsRateLimited(ip2) {
		t.Error("Expected IP2 third access to be rate limited")
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with very short window
	limiter := NewLimiter(3, 1)
	defer limiter.Close()
	if limiter.IsRateLimited("192.168.1.5") {
		t.Error("Expected first request to not be limited with short window")
	}

	// Test cleanup of old records
	limiter = NewLimiter(1, 1)
	defer limiter.Close()
	for i := 0; i < 200; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.IsRateLimited(ip)
	}
	time.Sleep(1100 * time.Millisecond)
	for i := 0; i < 5; i++ {
		ip := fmt.Sprintf("192.168.2.%d", i)
		limiter.IsRateLimited(ip)
	}
	if len(limiter.ipMap) != 5 {
		t.Errorf("Expected old records to be cleaned up, got %d records", len(limiter.ipMap))
	}
}

func TestBurstTraffic(t *testing.T) {
	limiter := NewLimiter(5, 1)
	defer limiter.Close()
	testIP := "192.168.1.6"

	// Simulate burst traffic
	for i := 0; i < 5; i++ {
		if limiter.IsRateLimited(testIP) {
			t.Errorf("Expected request %d to not be limited in burst", i+1)
		}
	}

	// Verify burst limit
	if !limiter.IsRateLimited(testIP) {
		t.Error("Expected request to be limited after burst")
	}

	// Wait for window reset
	time.Sleep(1100 * time.Millisecond)

	// Verify counter reset
	if limiter.IsRateLimited(testIP) {
		t.Error("Expected request to not be limited after window reset")
	}
}
