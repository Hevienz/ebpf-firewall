package main

import (
	"embed"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/ebpf"
	"github.com/danger-dream/ebpf-firewall/internal/metrics"
	"github.com/danger-dream/ebpf-firewall/internal/processor"
	"github.com/danger-dream/ebpf-firewall/internal/server"
	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
)

//go:embed web/dist
var Static embed.FS

func main() {
	if err := config.Init(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	config := config.GetConfig()
	data, _ := json.MarshalIndent(config, "", "  ")
	log.Printf("Current configuration:\n%s", string(data))

	pool := utils.NewElasticPool[*types.PacketInfo](utils.PoolConfig{
		QueueSize:  1024,
		MinWorkers: 3,
		MaxWorkers: int32(runtime.NumCPU() * 2),
	})

	collector := metrics.NewMetricsCollector()
	ebpfManager := ebpf.NewEBPFManager(pool)

	processor, err := processor.NewProcessor(pool, ebpfManager, collector)
	if err != nil {
		log.Fatalf("failed to start processor: %v", err)
	}

	if err := ebpfManager.Start(); err != nil {
		log.Fatalf("failed to start eBPF manager: %v", err)
	}

	pool.Start()

	appServer := server.New(ebpfManager, collector, processor)

	// priority: Try to serve local static files first
	distPath := filepath.Join(config.DataDir, "dist")
	if info, err := os.Stat(distPath); err == nil && info.IsDir() {
		log.Printf("Using local static files from: %s", distPath)
		appServer.ServeStaticDirectory(distPath)
	} else {
		if os.IsNotExist(err) {
			log.Printf("Local static directory not found, using embedded files")
		} else {
			log.Printf("Error accessing local static directory: %v, falling back to embedded files", err)
		}
		appServer.ServeEmbeddedFiles(Static)
	}

	appServer.HandleStatusNotFound()

	errChan := make(chan error, 1)
	go func() {
		if err := appServer.Start(); err != nil {
			errChan <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case err := <-errChan:
		log.Printf("server start failed: %v", err)
	case <-stop:
		log.Println("shutting down application...")
	}
	closeWithTimeout("appServer", appServer.Close, time.Second)
	closeWithTimeout("ebpfManager", ebpfManager.Close, time.Second)
	closeWithTimeout("processor", processor.Close, time.Second)
	closeWithTimeout("pool", pool.Close, time.Second)
	closeWithTimeout("collector", collector.Close, time.Second)
}

func closeWithTimeout(name string, fn func() error, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		start := time.Now()
		fn()
		log.Printf("Component %s closed in %v", name, time.Since(start))
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		log.Printf("Warning: %s close timeout after %v", name, timeout)
	}
}
