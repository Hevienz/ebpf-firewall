package middleware

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3/log"
)

// holds the count and time information for an IP address
type IPErrorCounter struct {
	IP        string `json:"ip"`
	Count     int    `json:"count"`
	FirstTime int64  `json:"first_time"`
	LastTime  int64  `json:"last_time"`
	ErrorType string `json:"error_type"`
}

// holds the error counter and block list for an IP address
type Block struct {
	// map of IP addresses to their error counters
	ErrorCounter map[string]*IPErrorCounter `json:"error_counter"`
	// map of IP addresses to their block status
	BlockList map[string]bool `json:"block_list"`
}

// security manager
type Security struct {
	// path to the block list file
	blockListPath string
	// maximum number of errors allowed per IP before blocking
	ipErrorThreshold int
	// time window in seconds for counting errors
	errorWindow int
	// block list
	block *Block
	// mutex for concurrent access
	mu sync.RWMutex
	// channel to signal the cleanup goroutine to stop
	channel chan struct{}
}

func NewSecurity(dataDir string, ipErrorThreshold int, errorWindow int) *Security {
	if ipErrorThreshold <= 0 {
		ipErrorThreshold = 10
	}
	if errorWindow <= 0 {
		errorWindow = 60 * 60 * 24
	}
	security := &Security{
		blockListPath:    filepath.Join(dataDir, "blacklist.json"),
		ipErrorThreshold: ipErrorThreshold,
		errorWindow:      errorWindow,
		block: &Block{
			ErrorCounter: make(map[string]*IPErrorCounter),
			BlockList:    make(map[string]bool),
		},
		channel: make(chan struct{}),
	}
	go security.cleanup()
	security.loadBlockList()
	return security
}

// cleanup the error counter
func (s *Security) cleanup() {
	ticker := time.NewTicker(time.Second * time.Duration(s.errorWindow))
	defer ticker.Stop()
	for {
		select {
		case <-s.channel:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now().Unix()
			hasChange := false
			for ip, record := range s.block.ErrorCounter {
				// if the last time of error is greater than the error window, delete the error counter
				if now-record.LastTime > int64(s.errorWindow) {
					delete(s.block.ErrorCounter, ip)
					hasChange = true
				}
			}
			// if there is any change, save the block list
			if hasChange {
				s.saveBlockList()
			}
			s.mu.Unlock()
		}
	}
}

// load the block list from the file
func (s *Security) loadBlockList() error {
	if s.blockListPath == "" {
		return nil
	}
	if _, err := os.Stat(s.blockListPath); os.IsNotExist(err) {
		return nil
	}
	blockList, err := os.ReadFile(s.blockListPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(blockList, s.block)
}

// save the block list to the file
func (s *Security) saveBlockList() error {
	if s.blockListPath == "" {
		return nil
	}
	blockList, err := json.Marshal(s.block)
	if err != nil {
		return err
	}
	return os.WriteFile(s.blockListPath, blockList, 0644)
}

// add a record to the error counter
func (s *Security) AddRecord(ip string, errorType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.block.BlockList[ip]; ok {
		return
	}
	now := time.Now().Unix()
	record, ok := s.block.ErrorCounter[ip]
	if !ok {
		record = &IPErrorCounter{
			IP:        ip,
			Count:     0,
			FirstTime: now,
		}
	}
	record.Count++
	record.LastTime = now
	record.ErrorType = errorType
	s.block.ErrorCounter[ip] = record
	// if the count of errors is greater than the threshold and the time window is within the error window, block the IP address
	if record.Count >= s.ipErrorThreshold && now-record.FirstTime <= int64(s.errorWindow) {
		s.block.BlockList[ip] = true
		log.Warnf("ip %s is blocked", ip)
		s.saveBlockList()
	}
}

// check if an IP address is blocked
func (s *Security) IsBlocked(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.block.BlockList[ip]; ok {
		return true
	}
	return false
}

func (s *Security) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.block.ErrorCounter = make(map[string]*IPErrorCounter)
	s.block.BlockList = make(map[string]bool)
	s.saveBlockList()
}
