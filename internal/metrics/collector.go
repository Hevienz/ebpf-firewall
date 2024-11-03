package metrics

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
)

const (
	DefaultRetentionDays = 30
	DayFormat            = "20060102"

	// Dimension keys for different metric categories
	dimensionCountry = "country"
	dimensionCity    = "city"
	dimensionPort    = "dst_port"
	dimensionEthType = "eth_type"
	dimensionIPProto = "ip_proto"
	dimensionMatch   = "match"
)

// geo location of a packet
type GeoLocation struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

// protocol of a packet
type Protocol struct {
	EthType types.EthernetType `json:"eth_type"`
	IPProto types.IPProtocol   `json:"ip_proto"`
}

// basic metrics data for network traffic
type TrafficMetrics struct {
	TotalPackets int64 `json:"total_packets"`
	TotalBytes   int64 `json:"total_bytes"`
	FirstSeenAt  int64 `json:"first_seen_at"`
	LastSeenAt   int64 `json:"last_seen_at"`
}

// statistics of a dimension
type Statistics struct {
	Key            string `json:"key"`
	TrafficMetrics `json:",inline"`
}

// source of a packet
type Source struct {
	MAC      string      `json:"src_mac"`
	IP       string      `json:"src_ip"`
	Port     uint16      `json:"src_port"`
	Location GeoLocation `json:"location"`
}

// destination of a packet
type Destination struct {
	MAC  string `json:"target_mac"`
	IP   string `json:"target_ip"`
	Port uint16 `json:"target_port"`
	Protocol
}

// statistics of a source
type SourceStatistic struct {
	Key            string `json:"key"`
	Source         `json:",inline"`
	TrafficMetrics `json:",inline"`
	Target         map[string]TargetStatistic `json:"target"`
}

// statistics of a source
type SourceStatisticResult struct {
	Key            string `json:"key"`
	Source         `json:",inline"`
	TrafficMetrics `json:",inline"`
	Targets        int64 `json:"targets"`
}

// statistics of a target
type TargetStatistic struct {
	Key            string `json:"key"`
	Destination    `json:",inline"`
	TrafficMetrics `json:",inline"`
}

// summary of collected metrics
type MetricsSummary struct {
	TotalPackets int64                             `json:"total_packets"`
	TotalBytes   int64                             `json:"total_bytes"`
	Day          map[string]Statistics             `json:"day"`
	Statistics   map[string]map[string]*Statistics `json:"statistics"`
	Source       map[string]SourceStatistic        `json:"source"`
}

// MetricsCollector handles the collection and aggregation of network metrics
type MetricsCollector struct {
	summary MetricsSummary
	mu      sync.RWMutex
	done    chan struct{}
	storage *MetricsStorage
}

// NewMetricsCollector creates and initializes a new metrics collector instance
func NewMetricsCollector() *MetricsCollector {
	storage := NewMetricsStorage(config.GetConfig().DataDir)

	var summary MetricsSummary
	if metrics, err := storage.Load(); err == nil && metrics != nil {
		summary = *metrics
	} else {
		summary = MetricsSummary{
			Day:        make(map[string]Statistics),
			Statistics: make(map[string]map[string]*Statistics),
			Source:     make(map[string]SourceStatistic),
		}
	}

	metricsCollector := &MetricsCollector{
		summary: summary,
		done:    make(chan struct{}),
		storage: storage,
	}

	go metricsCollector.autoCleanup()
	go metricsCollector.autoPersist()
	return metricsCollector
}

// periodically persist metrics data
func (mc *MetricsCollector) autoPersist() {
	ticker := time.NewTicker(time.Minute * time.Duration(config.GetConfig().MetricsPersistInterval))
	defer ticker.Stop()
	for {
		select {
		case <-mc.done:
			return
		case <-ticker.C:
			if err := mc.storage.Save(&mc.summary); err != nil {
				log.Printf("保存指标数据失败: %v", err)
			}
		}
	}
}

// periodically removes stale metrics data
func (mc *MetricsCollector) autoCleanup() {
	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()
	for {
		select {
		case <-mc.done:
			return
		case <-ticker.C:
			mc.CleanupStaleMetrics(time.Hour * time.Duration(config.GetConfig().RetentionHours))
		}
	}
}

func (mc *MetricsCollector) CleanupStaleMetrics(retention time.Duration) {
	now := time.Now().Unix()

	for _, stats := range mc.summary.Statistics {
		for key, stat := range stats {
			if now-stat.LastSeenAt > int64(retention.Seconds()) {
				delete(stats, key)
			}
		}
	}
	for _, source := range mc.summary.Source {
		if now-source.LastSeenAt > int64(retention.Seconds()) {
			delete(mc.summary.Source, source.Key)
		}
	}
}

// processes metrics for a single packet
func (mc *MetricsCollector) CollectPacket(packet *types.Packet) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.updateSummaryMetrics(packet)
	mc.updateDimensionMetrics(packet)
	mc.updateSourceMetrics(packet)
}

func (mc *MetricsCollector) updateSummaryMetrics(packet *types.Packet) {
	mc.summary.TotalPackets++
	mc.summary.TotalBytes += int64(packet.Size)
	day := time.Now().Format(DayFormat)
	dayData, ok := mc.summary.Day[day]
	if !ok {
		dayData = Statistics{
			Key: day,
		}
	}
	dayData.TotalPackets++
	dayData.TotalBytes += int64(packet.Size)
	mc.summary.Day[day] = dayData
}

// updates metrics for dimensions
func (mc *MetricsCollector) updateDimensionMetrics(packet *types.Packet) {
	mc.updateMetrics(dimensionCountry, packet.Country, packet.Size)
	mc.updateMetrics(dimensionCity, packet.City, packet.Size)
	if packet.DstPort > 0 {
		mc.updateMetrics(dimensionPort, fmt.Sprint(packet.DstPort), packet.Size)
	}
	mc.updateMetrics(dimensionEthType, packet.EthType.String(), packet.Size)
	mc.updateMetrics(dimensionIPProto, packet.IPProto.String(), packet.Size)

	if packet.MatchType != types.NoMatch {
		key := packet.SrcIP
		if packet.MatchType == types.MatchByMAC {
			key = packet.SrcMAC
		}
		mc.updateMetrics(dimensionMatch, key, packet.Size)
	}
}

func (mc *MetricsCollector) updateSourceMetrics(packet *types.Packet) {
	sourceKey := utils.MD5(fmt.Sprintf("%s:%s", packet.SrcMAC, packet.SrcIP))
	sourceData, ok := mc.summary.Source[sourceKey]
	if !ok {
		// create new source metrics if not exists
		sourceData = SourceStatistic{
			Key: sourceKey,
			Source: Source{
				IP:  packet.SrcIP,
				MAC: packet.SrcMAC,
				Location: GeoLocation{
					Country: packet.Country,
					City:    packet.City,
				},
			},
			TrafficMetrics: TrafficMetrics{
				TotalPackets: 0,
				TotalBytes:   0,
				FirstSeenAt:  time.Now().Unix(),
				LastSeenAt:   time.Now().Unix(),
			},
			Target: make(map[string]TargetStatistic),
		}
	}
	sourceData.TrafficMetrics.TotalPackets++
	sourceData.TrafficMetrics.TotalBytes += int64(packet.Size)
	sourceData.TrafficMetrics.LastSeenAt = time.Now().Unix()

	targetKey := utils.MD5(fmt.Sprintf("%s:%s:%d:%s:%s", packet.DstMAC, packet.DstIP, packet.DstPort, packet.EthType, packet.IPProto))
	targetData, ok := sourceData.Target[targetKey]
	if !ok {
		// create new target metrics if not exists
		targetData = TargetStatistic{
			Key: targetKey,
			Destination: Destination{
				MAC:  packet.DstMAC,
				IP:   packet.DstIP,
				Port: packet.DstPort,
				Protocol: Protocol{
					EthType: packet.EthType,
					IPProto: packet.IPProto,
				},
			},
			TrafficMetrics: TrafficMetrics{
				TotalPackets: 0,
				TotalBytes:   0,
				FirstSeenAt:  time.Now().Unix(),
				LastSeenAt:   time.Now().Unix(),
			},
		}
	}
	targetData.TrafficMetrics.TotalPackets++
	targetData.TrafficMetrics.TotalBytes += int64(packet.Size)
	targetData.TrafficMetrics.LastSeenAt = time.Now().Unix()
	sourceData.Target[targetKey] = targetData
	mc.summary.Source[sourceKey] = sourceData
}

// updates metrics for a given dimension and key
func (mc *MetricsCollector) updateMetrics(dimension string, key string, size uint32) {
	if dimension == "" || key == "" {
		return
	}
	// initialize dimension if not exists
	if _, exists := mc.summary.Statistics[dimension]; !exists {
		mc.summary.Statistics[dimension] = make(map[string]*Statistics)
	}
	// update metrics if key exists
	if stat, exists := mc.summary.Statistics[dimension][key]; exists {
		stat.TrafficMetrics.TotalPackets++
		stat.TrafficMetrics.TotalBytes += int64(size)
		stat.TrafficMetrics.LastSeenAt = time.Now().Unix()
	} else {
		// create new metrics if key does not exist
		mc.summary.Statistics[dimension][key] = &Statistics{
			Key: key,
			TrafficMetrics: TrafficMetrics{
				TotalPackets: 1,
				TotalBytes:   int64(size),
				FirstSeenAt:  time.Now().Unix(),
				LastSeenAt:   time.Now().Unix(),
			},
		}
	}
}

// summary report of collected metrics
type MetricsReport struct {
	TotalPackets int64                   `json:"total_packets"`
	TotalBytes   int64                   `json:"total_bytes"`
	Day          []Statistics            `json:"day"`
	Dimension    map[string][]Statistics `json:"dimension"`
}

// GenerateReport creates a summary report of collected metrics
// top parameter determines the maximum number of entries to include in each category
func (mc *MetricsCollector) GenerateReport(top int) MetricsReport {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	result := MetricsReport{
		TotalPackets: mc.summary.TotalPackets,
		TotalBytes:   mc.summary.TotalBytes,
	}

	// collect day statistics
	dayList := make([]Statistics, 0, len(mc.summary.Day))
	for _, day := range mc.summary.Day {
		dayList = append(dayList, day)
	}
	if len(dayList) > DefaultRetentionDays {
		result.Day = dayList[len(dayList)-DefaultRetentionDays:]
	} else {
		result.Day = dayList
	}

	// collect dimension statistics
	result.Dimension = make(map[string][]Statistics, len(mc.summary.Statistics))
	for category, statsMap := range mc.summary.Statistics {
		statsList := make([]Statistics, 0, len(statsMap))
		for _, stat := range statsMap {
			statsList = append(statsList, *stat)
		}
		sort.Slice(statsList, func(i, j int) bool {
			return statsList[i].TrafficMetrics.TotalPackets > statsList[j].TrafficMetrics.TotalPackets
		})
		if len(statsList) > top {
			result.Dimension[category] = statsList[:top]
		} else {
			result.Dimension[category] = statsList
		}
	}
	return result
}

type SourcePage struct {
	Total int                     `json:"total"`
	Items []SourceStatisticResult `json:"items"`
}

func (mc *MetricsCollector) GetSources(page int, pageSize int, order string, sortDir string) SourcePage {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	result := make([]SourceStatisticResult, 0, len(mc.summary.Source))
	for _, source := range mc.summary.Source {
		result = append(result, SourceStatisticResult{
			Key:            source.Key,
			Source:         source.Source,
			TrafficMetrics: source.TrafficMetrics,
			Targets:        int64(len(source.Target)),
		})
	}
	getField := func(s SourceStatisticResult) int64 {
		switch order {
		case "total_packets":
			return s.TotalPackets
		case "total_bytes":
			return s.TotalBytes
		case "first_seen_at":
			return s.FirstSeenAt
		case "targets":
			return s.Targets
		default: // last_seen_at
			return s.LastSeenAt
		}
	}
	if sortDir == "" {
		sortDir = "desc"
	}
	sort.Slice(result, func(i, j int) bool {
		if sortDir == "desc" {
			return getField(result[i]) > getField(result[j])
		}
		return getField(result[i]) < getField(result[j])
	})
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	return SourcePage{
		Total: len(result),
		Items: result[start:end],
	}
}

type TargetPage struct {
	Total int               `json:"total"`
	Items []TargetStatistic `json:"items"`
}

func (mc *MetricsCollector) GetTargets(sourceId string, page int, pageSize int, order string, sortDir string) TargetPage {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	sourceData, ok := mc.summary.Source[sourceId]
	if !ok {
		return TargetPage{
			Total: 0,
			Items: []TargetStatistic{},
		}
	}
	result := make([]TargetStatistic, 0, len(sourceData.Target))
	for _, target := range sourceData.Target {
		result = append(result, target)
	}
	getField := func(s TargetStatistic) int64 {
		switch order {
		case "total_packets":
			return s.TotalPackets
		case "total_bytes":
			return s.TotalBytes
		case "first_seen_at":
			return s.FirstSeenAt
		case "eth_type":
			return int64(s.EthType)
		case "ip_proto":
			return int64(s.IPProto)
		default: // last_seen_at
			return s.LastSeenAt
		}
	}
	if sortDir == "" {
		sortDir = "desc"
	}
	sort.Slice(result, func(i, j int) bool {
		if sortDir == "desc" {
			return getField(result[i]) > getField(result[j])
		}
		return getField(result[i]) < getField(result[j])
	})
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	return TargetPage{
		Total: len(result),
		Items: result[start:end],
	}
}

func (mc *MetricsCollector) Close() error {
	close(mc.done)
	if mc.storage != nil {
		if err := mc.storage.Save(&mc.summary); err != nil {
			return err
		}
	}
	return nil
}
