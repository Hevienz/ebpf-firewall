package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type MetricsStorage struct {
	filePath string
	mu       sync.RWMutex
}

func NewMetricsStorage(dataDir string) *MetricsStorage {
	return &MetricsStorage{
		filePath: filepath.Join(dataDir, "metrics.json"),
	}
}

func (ms *MetricsStorage) Save(metrics *MetricsSummary) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(ms.filePath, data, 0644)
}

func (ms *MetricsStorage) Load() (*MetricsSummary, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	data, err := os.ReadFile(ms.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var metrics MetricsSummary
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

func (ms *MetricsStorage) DeleteMetrics() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return os.Remove(ms.filePath)
}
