package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/threatintel"
)

const (
	defaultCleanupInterval = time.Second * 15
	defaultBlockDuration   = time.Hour * 24 * 7
	defaultConfigFile      = "processor.json"
)

type BlockSourceType uint8

const (
	BlockSourceTypeUser BlockSourceType = 1
	BlockSourceTypeIntel
	BlockSourceTypeAnalyzer
)

type MatchActionMode uint8

const (
	MatchActionModeMonitor MatchActionMode = 1
	MatchActionModeBlock
	MatchActionModeThreshold
)

type BlockRule struct {
	ID         string          `json:"id"`
	Value      string          `json:"value"` // IP/CIDR、MAC
	Note       string          `json:"note"`
	Source     BlockSourceType `json:"source"`
	CreateTime int64           `json:"create_time"`
	Enabled    bool            `json:"enabled"`
	ExpireTime int64           `json:"expire_time"` // Ignore when it's zero.
	Extra      map[string]any  `json:"extra"`
}

type ProcessorConfig struct {
	CleanupInterval time.Duration `json:"cleanup_interval"`
	Blocklist       struct {
		// default block duration
		DefaultBlockDuration time.Duration `json:"default_block_duration"`
		Rules                []BlockRule   `json:"rules"`
	} `json:"blocklist"`

	ThreatIntel struct {
		// Whether to ignore the check of local network.
		IgnoreLocalNetwork bool            `json:"ignore_local_network"`
		MatchMode          MatchActionMode `json:"match_mode"`
		// The number of times a single IP matches within the specified window period before it is added to the blacklist.
		MatchThreshold int           `json:"match_threshold"`
		MatchWindow    time.Duration `json:"match_window"`
		// The duration for which the IP is blocked after it is matched.
		BlockDuration time.Duration                       `json:"block_duration"`
		Feeds         map[string]threatintel.FeedMetadata `json:"feeds"`
	} `json:"threat_intel"`
}

func (p *Processor) getDefaultConfig() *ProcessorConfig {
	config := &ProcessorConfig{}

	// default config
	config.CleanupInterval = defaultCleanupInterval
	config.Blocklist.DefaultBlockDuration = defaultBlockDuration
	config.Blocklist.Rules = make([]BlockRule, 0)

	config.ThreatIntel.MatchMode = MatchActionModeThreshold
	config.ThreatIntel.IgnoreLocalNetwork = true
	config.ThreatIntel.MatchThreshold = 3
	config.ThreatIntel.MatchWindow = time.Hour * 24
	config.ThreatIntel.BlockDuration = time.Hour * 24 * 7
	config.ThreatIntel.Feeds = p.threatAggregator.GenerateFeedsMetadata()
	return config
}

func (p *Processor) getConfig() *ProcessorConfig {
	return p.config.Load().(*ProcessorConfig)
}

func (p *Processor) updateConfig(updater func(*ProcessorConfig) error) error {
	newConfig := *p.getConfig()
	if err := updater(&newConfig); err != nil {
		return err
	}
	p.config.Store(&newConfig)
	return nil
}

func (p *Processor) loadConfig() error {
	configPath := filepath.Join(p.dataDir, defaultConfigFile)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := p.getDefaultConfig()
		p.config.Store(config)
		if err := p.saveConfig(); err != nil {
			return err
		}
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	config := &ProcessorConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return err
	}

	p.config.Store(config)
	return nil
}

func (p *Processor) saveConfig() error {
	data, err := json.MarshalIndent(*p.getConfig(), "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(p.dataDir, defaultConfigFile), data, 0644)
}
