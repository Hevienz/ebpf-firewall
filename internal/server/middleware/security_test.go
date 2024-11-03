package middleware

import (
	"os"
	"testing"
	"time"
)

func TestSecurity(t *testing.T) {
	tmpFile := "test_blocklist.json"
	defer os.Remove(tmpFile)

	// Test Case 1: Basic initialization and first access
	t.Run("Basic initialization", func(t *testing.T) {
		sec := NewSecurity(tmpFile, 3, 5)
		defer sec.Clear()
		testIP := "192.168.1.1"

		if sec.IsBlocked(testIP) {
			t.Error("Expected first access to not be blocked")
		}

		// Add records within threshold
		for i := 0; i < 2; i++ {
			sec.AddRecord(testIP, "login_failed")
			if sec.IsBlocked(testIP) {
				t.Errorf("Expected access %d to not be blocked", i+1)
			}
		}

		// Exceed threshold
		sec.AddRecord(testIP, "login_failed")
		if !sec.IsBlocked(testIP) {
			t.Error("Expected IP to be blocked after exceeding threshold")
		}
	})

	// Test Case 2: Test window expiration
	t.Run("Window expiration", func(t *testing.T) {
		sec := NewSecurity(tmpFile, 3, 1) // 1 second window
		defer sec.Clear()
		testIP := "192.168.1.2"
		// Add two records
		sec.AddRecord(testIP, "login_failed")
		sec.AddRecord(testIP, "login_failed")

		// Wait for window to expire
		time.Sleep(1100 * time.Millisecond)

		// Verify records are cleaned
		if _, ok := sec.block.ErrorCounter[testIP]; !ok {
			t.Error("Expected records to be cleaned after window expiration")
		}
	})

	// Test Case 3: Concurrent access
	t.Run("Concurrent access", func(t *testing.T) {
		sec := NewSecurity(tmpFile, 100, 5)
		defer sec.Clear()
		testIP := "192.168.1.3"

		done := make(chan bool)
		for i := 0; i < 50; i++ {
			go func() {
				sec.AddRecord(testIP, "login_failed")
				done <- true
			}()
		}

		for i := 0; i < 50; i++ {
			<-done
		}

		if sec.block.ErrorCounter[testIP].Count != 50 {
			t.Errorf("Expected error count to be 50, got %d", sec.block.ErrorCounter[testIP].Count)
		}
	})

	// Test Case 4: Multiple IPs
	t.Run("Multiple IPs", func(t *testing.T) {
		sec := NewSecurity(tmpFile, 3, 5)
		defer sec.Clear()
		ip1 := "192.168.1.4"
		ip2 := "192.168.1.5"

		// Test IP1
		for i := 0; i < 2; i++ {
			sec.AddRecord(ip1, "login_failed")
			if sec.IsBlocked(ip1) {
				t.Errorf("Expected IP1 access %d to not be blocked", i+1)
			}
		}
		sec.AddRecord(ip1, "login_failed")
		if !sec.IsBlocked(ip1) {
			t.Error("Expected IP1 to be blocked")
		}

		// Test IP2 (should not be affected by IP1)
		if sec.IsBlocked(ip2) {
			t.Error("Expected IP2 to not be blocked initially")
		}
	})

	// Test Case 5: Persistence
	t.Run("Persistence", func(t *testing.T) {
		sec1 := NewSecurity(tmpFile, 3, 5)
		defer sec1.Clear()
		testIP := "192.168.1.6"

		// Block an IP
		for i := 0; i < 3; i++ {
			sec1.AddRecord(testIP, "login_failed")
		}

		// Create new instance and verify blocked status
		sec2 := NewSecurity(tmpFile, 3, 5)
		defer sec2.Clear()
		if !sec2.IsBlocked(testIP) {
			t.Error("Expected IP to remain blocked after reload")
		}
	})

	// Test Case 6: Edge Cases
	t.Run("Edge cases", func(t *testing.T) {
		sec := NewSecurity(tmpFile, 1, 1)
		defer sec.Clear()
		testIP := "192.168.1.7"

		// Test immediate blocking
		sec.AddRecord(testIP, "login_failed")
		if !sec.IsBlocked(testIP) {
			t.Error("Expected immediate blocking with threshold 1")
		}

		// Test adding record to already blocked IP
		sec.AddRecord(testIP, "login_failed")
		if sec.block.ErrorCounter[testIP].Count != 1 {
			t.Error("Expected counter to not increase for blocked IP")
		}
	})
}
