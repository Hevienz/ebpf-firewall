package threatintel

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/robfig/cron/v3"
	"golang.org/x/exp/maps"

	"github.com/danger-dream/ebpf-firewall/internal/threatintel/iptrie"
	"github.com/danger-dream/ebpf-firewall/internal/threatintel/provider"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
)

type FeedMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Schedule    string            `json:"schedule"`
	Enabled     bool              `json:"enabled"`
	Params      map[string]string `json:"params"`
}

type ThreatFeed interface {
	Name() string
	Description() string
	Schedule() string
	Fetch(params map[string]string) ([]string, error)
	DefaultParams() map[string]string
}

type Aggregator struct {
	dataDir  string
	cron     *cron.Cron
	entryIDs map[string]cron.EntryID
	trie     *iptrie.IPTrie
	feeds    map[string]ThreatFeed
	metadata *sync.Map
	mu       sync.RWMutex
}

func NewAggregator(dataDir string) (*Aggregator, error) {
	dir := filepath.Join(dataDir, "threatintel")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	agg := &Aggregator{
		dataDir:  dir,
		cron:     cron.New(),
		entryIDs: make(map[string]cron.EntryID),
		trie:     iptrie.NewIPTrie(),
		feeds:    make(map[string]ThreatFeed),
		metadata: &sync.Map{},
	}
	if err := agg.registerFeed(&provider.AbuseIPDB{}); err != nil {
		return nil, err
	}
	if err := agg.registerFeed(&provider.Spamhaus{}); err != nil {
		return nil, err
	}
	return agg, nil
}

func (a *Aggregator) registerFeed(feed ThreatFeed) error {
	if feed == nil {
		return errors.New("feed cannot be nil")
	}

	schedule := feed.Schedule()
	if schedule == "" {
		return errors.New("feed schedule cannot be empty")
	}

	if _, err := cron.ParseStandard(schedule); err != nil {
		return fmt.Errorf("invalid schedule expression: %v", err)
	}

	name := strings.ToLower(feed.Name())
	a.feeds[name] = feed
	return nil
}

func (a *Aggregator) GenerateFeedsMetadata() map[string]FeedMetadata {
	metadata := make(map[string]FeedMetadata)
	for name, feed := range a.feeds {
		metadata[name] = FeedMetadata{
			Name:        feed.Name(),
			Description: feed.Description(),
			Schedule:    feed.Schedule(),
			Enabled:     false,
			Params:      feed.DefaultParams(),
		}
	}
	return metadata
}

func (a *Aggregator) Initialize(metadata map[string]FeedMetadata) error {
	for name, info := range metadata {
		a.metadata.Store(name, &info)
		if info.Enabled {
			if err := a.schedule(name, info.Schedule); err != nil {
				return fmt.Errorf("failed to schedule feed %s: %v", name, err)
			}
		}
	}
	a.cron.Start()
	return nil
}

func (a *Aggregator) getIntelligenceFilename(name string) string {
	return filepath.Join(a.dataDir, fmt.Sprintf("%s.txt", name))
}

func (a *Aggregator) syncFeed(name string) {
	source, exists := a.feeds[name]
	if !exists {
		return
	}
	infoVal, exists := a.metadata.Load(name)
	info, _ := infoVal.(*FeedMetadata)
	if !exists || info == nil || !info.Enabled {
		return
	}

	ips, err := source.Fetch(info.Params)
	if err != nil {
		log.Printf("Failed to fetch data from feed %s: %v", name, err)
		return
	}
	if len(ips) == 0 {
		log.Printf("No indicators retrieved from feed %s", name)
		return
	}

	validIPs := make([]string, 0, len(ips))
	for _, ip := range ips {
		if utils.ParseStringToIPType(ip) != utils.IPTypeUnknown {
			validIPs = append(validIPs, ip)
		}
	}
	ips = validIPs

	if len(ips) == 0 {
		log.Printf("No valid indicators retrieved from feed %s", name)
		return
	}

	log.Printf("Successfully retrieved %d indicators from feed %s", len(ips), name)

	filename := a.getIntelligenceFilename(name)
	if err := os.WriteFile(filename, []byte(strings.Join(ips, "\n")), 0644); err != nil {
		log.Printf("Failed to save data for feed %s: %v", name, err)
		return
	}

	a.aggregateIndicators()
}

func (a *Aggregator) aggregateIndicators() {
	trie := iptrie.NewIPTrie()
	total := 0
	a.metadata.Range(func(key, value interface{}) bool {
		info, _ := value.(*FeedMetadata)
		if !info.Enabled {
			return true
		}
		filename := a.getIntelligenceFilename(info.Name)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return true
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Printf("Failed to read feed file %s: %v", filename, err)
			return true
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if err := trie.Insert(line); err == nil {
				total++
			}
		}
		return true
	})
	a.mu.Lock()
	a.trie = trie
	a.mu.Unlock()
	log.Printf("threatintel aggregate indicators: %d", total)
}

func (a *Aggregator) Close() {
	a.cron.Stop()
}

func (a *Aggregator) schedule(name, schedule string) error {
	// remove the old schedule
	a.mu.Lock()
	if id, exists := a.entryIDs[name]; exists {
		a.cron.Remove(id)
		delete(a.entryIDs, name)
	}
	id, err := a.cron.AddFunc(schedule, func() {
		a.syncFeed(name)
	})
	if err != nil {
		a.mu.Unlock()
		return fmt.Errorf("failed to schedule task: %v", err)
	}
	a.entryIDs[name] = id
	a.mu.Unlock()
	a.syncFeed(name)
	return nil
}

func (a *Aggregator) disableFeed(name string) {
	a.mu.Lock()
	// remove the old schedule
	if id, exists := a.entryIDs[name]; exists {
		a.cron.Remove(id)
		delete(a.entryIDs, name)
	}
	a.mu.Unlock()
	if _, exists := a.metadata.Load(name); exists {
		filename := a.getIntelligenceFilename(name)
		if _, err := os.Stat(filename); err == nil {
			os.Remove(filename)
		}
		a.metadata.Delete(name)
	}
	a.aggregateIndicators()
}

func (a *Aggregator) UpdateFeedMetadata(name string, metadata *FeedMetadata) error {
	name = strings.ToLower(name)

	if _, exists := a.feeds[name]; !exists {
		return fmt.Errorf("feed not found: %s", name)
	}

	if metadata.Schedule != "" {
		if _, err := cron.ParseStandard(metadata.Schedule); err != nil {
			return fmt.Errorf("invalid schedule expression: %v", err)
		}
	}
	currentVal, exists := a.metadata.Load(name)
	if !exists {
		return fmt.Errorf("feed metadata not found: %s", name)
	}
	current, _ := currentVal.(*FeedMetadata)
	oldEnabled := current.Enabled
	a.metadata.Store(name, metadata)

	switch {
	case !oldEnabled && metadata.Enabled:
		if err := a.schedule(name, metadata.Schedule); err != nil {
			a.metadata.Store(name, current)
			return err
		}

	case oldEnabled && !metadata.Enabled:
		a.disableFeed(name)
	case metadata.Enabled:
		// update the feed
		if current.Schedule != metadata.Schedule {
			if err := a.schedule(name, metadata.Schedule); err != nil {
				a.metadata.Store(name, current)
				return err
			}
		} else if !maps.Equal(current.Params, metadata.Params) {
			a.syncFeed(name)
		}
	}
	return nil
}

func (a *Aggregator) GetFeedMetadata(name string) *FeedMetadata {
	infoVal, exists := a.metadata.Load(name)
	if !exists {
		return nil
	}
	info, _ := infoVal.(*FeedMetadata)
	return info
}

func (a *Aggregator) GetFeedsMetadata() map[string]*FeedMetadata {
	metadata := make(map[string]*FeedMetadata)
	a.metadata.Range(func(key, value interface{}) bool {
		info, _ := value.(*FeedMetadata)
		metadata[key.(string)] = info
		return true
	})
	return metadata
}

func (a *Aggregator) Contains(ip string) bool {
	if ip == "" || a.trie == nil || a.trie.Size() < 1 {
		return false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.trie.Contains(ip)
}
