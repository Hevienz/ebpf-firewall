package utils

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestElasticPool_BasicFunctionality(t *testing.T) {
	config := PoolConfig{
		QueueSize:     100,
		MinWorkers:    2,
		MaxWorkers:    10,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
		BackoffTime:   5 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)

	processedCount := atomic.Int32{}

	pool.SetProcessor(func(task int) {
		processedCount.Add(1)
		time.Sleep(10 * time.Millisecond)
	})

	totalTasks := int32(1000)
	var producedCount atomic.Int32

	pool.SetProducer(func(submit func(int)) {
		for i := 0; i < int(totalTasks); i++ {
			if producedCount.Load() >= totalTasks {
				return
			}
			submit(i)
			producedCount.Add(1)
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		if processedCount.Load() >= totalTasks {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("Timeout waiting for tasks to complete")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	if processedCount.Load() != totalTasks {
		t.Errorf("Expected %d tasks to be processed, got %d", totalTasks, processedCount.Load())
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("Failed to shutdown pool: %v", err)
	}
}

func TestElasticPool_WorkerScaling(t *testing.T) {
	config := PoolConfig{
		QueueSize:     10,
		MinWorkers:    1,
		MaxWorkers:    5,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
		BackoffTime:   5 * time.Millisecond,
	}

	pool := NewElasticPool[struct{}](config)

	var maxWorkers atomic.Int32
	var wg sync.WaitGroup
	wg.Add(100)

	pool.SetProcessor(func(task struct{}) {
		defer wg.Done()
		currentWorkers := pool.workerCount.Load()
		if maxWorkers.Load() < currentWorkers {
			maxWorkers.Store(currentWorkers)
		}
		time.Sleep(100 * time.Millisecond)
	})

	pool.SetProducer(func(submit func(struct{})) {
		for i := 0; i < 100; i++ {
			submit(struct{}{})
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout waiting for tasks to complete")
	}

	if maxWorkers.Load() != config.MaxWorkers {
		t.Errorf("Expected max workers to reach %d, got %d", config.MaxWorkers, maxWorkers.Load())
	}

	time.Sleep(config.IdleTimeout * 2)

	if err := pool.Close(); err != nil {
		t.Fatalf("Failed to shutdown pool: %v", err)
	}
}

func TestElasticPool_GracefulShutdown(t *testing.T) {
	config := PoolConfig{
		QueueSize:     100,
		MinWorkers:    2,
		MaxWorkers:    5,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
		BackoffTime:   5 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)

	var wg sync.WaitGroup
	processedTasks := make(map[int]bool)
	var mu sync.Mutex

	pool.SetProcessor(func(task int) {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		processedTasks[task] = true
		mu.Unlock()
		wg.Done()
	})

	totalTasks := 10
	wg.Add(totalTasks)

	pool.SetProducer(func(submit func(int)) {
		for i := 0; i < totalTasks; i++ {
			submit(i)
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	wg.Wait()

	if err := pool.Close(); err != nil {
		t.Fatalf("Failed to shutdown pool: %v", err)
	}

	if len(processedTasks) != totalTasks {
		t.Errorf("Expected %d tasks to be processed, got %d", totalTasks, len(processedTasks))
	}
}

func TestElasticPool_ErrorHandling(t *testing.T) {
	config := PoolConfig{
		QueueSize:     10,
		MinWorkers:    2,
		MaxWorkers:    5,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)

	if err := pool.Start(); err == nil {
		t.Error("Expected error when starting pool without producer")
	}

	pool.SetProducer(func(submit func(int)) {})
	if err := pool.Start(); err == nil {
		t.Error("Expected error when starting pool without processor")
	}
}

func TestElasticPool_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      PoolConfig
		expectError bool
	}{
		{
			name: "Zero values should be set to defaults",
			config: PoolConfig{
				QueueSize:  0,
				MinWorkers: 0,
				MaxWorkers: 0,
			},
		},
		{
			name: "Valid custom config",
			config: PoolConfig{
				QueueSize:     100,
				MinWorkers:    2,
				MaxWorkers:    10,
				ScaleInterval: 50 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewElasticPool[int](tt.config)

			if tt.config.QueueSize == 0 && cap(pool.taskQueue) != 1024 {
				t.Errorf("Expected default queue size 1024, got %d", cap(pool.taskQueue))
			}

			if tt.config.MinWorkers == 0 && pool.config.MinWorkers != 1 {
				t.Errorf("Expected default min workers 1, got %d", pool.config.MinWorkers)
			}
		})
	}
}

func TestElasticPool_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	config := PoolConfig{
		QueueSize:     1000,
		MinWorkers:    5,
		MaxWorkers:    50,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   1 * time.Second,
		BackoffTime:   5 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)

	var processed, produced atomic.Int64
	var maxWorkers atomic.Int32

	pool.SetProcessor(func(task int) {
		processed.Add(1)
		current := pool.workerCount.Load()
		for {
			max := maxWorkers.Load()
			if current <= max {
				break
			}
			if maxWorkers.CompareAndSwap(max, current) {
				break
			}
		}
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	})

	totalTasks := int64(50000)
	pool.SetProducer(func(submit func(int)) {
		for i := 0; produced.Load() < totalTasks; i++ {
			submit(i)
			produced.Add(1)
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Stress test timeout")
		case <-ticker.C:
			if processed.Load() == produced.Load() {
				if maxWorkers.Load() > config.MaxWorkers {
					t.Errorf("Max workers exceeded limit: got %d, want <= %d",
						maxWorkers.Load(), config.MaxWorkers)
				}
				return
			}
		}
	}
}

func TestElasticPool_TaskPanic(t *testing.T) {
	config := PoolConfig{
		QueueSize:     10,
		MinWorkers:    2,
		MaxWorkers:    5,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)
	var processed atomic.Int32
	var panics atomic.Int32

	pool.SetProcessor(func(task int) {
		if task%2 == 0 {
			panics.Add(1)
			panic("simulated panic")
		}
		processed.Add(1)
		time.Sleep(10 * time.Millisecond)
	})

	totalTasks := 10
	pool.SetProducer(func(submit func(int)) {
		for i := 0; i < totalTasks; i++ {
			submit(i)
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for processed.Load()+panics.Load() < int32(totalTasks) {
		select {
		case <-deadline:
			t.Fatal("Test timeout")
		case <-ticker.C:
			continue
		}
	}

	if processed.Load() != int32(totalTasks/2) {
		t.Errorf("Expected %d tasks to be processed, got %d",
			totalTasks/2, processed.Load())
	}

	if panics.Load() != int32(totalTasks/2) {
		t.Errorf("Expected %d panics, got %d",
			totalTasks/2, panics.Load())
	}
}

func TestElasticPool_ConcurrentProducers(t *testing.T) {
	config := PoolConfig{
		QueueSize:     100,
		MinWorkers:    2,
		MaxWorkers:    10,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)
	var processed atomic.Int32
	var produced atomic.Int32

	pool.SetProcessor(func(task int) {
		processed.Add(1)
		time.Sleep(time.Millisecond)
	})

	totalTasks := 500
	var wg sync.WaitGroup
	numGoroutines := 5
	tasksPerGoroutine := totalTasks / numGoroutines

	pool.SetProducer(func(submit func(int)) {
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(offset int) {
				defer wg.Done()
				for j := 0; j < tasksPerGoroutine; j++ {
					taskID := offset*tasksPerGoroutine + j
					submit(taskID)
					produced.Add(1)
				}
			}(i)
		}

		wg.Wait()
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("Timeout waiting for tasks to complete. Processed: %d, Produced: %d",
				processed.Load(), produced.Load())
		case <-ticker.C:
			if processed.Load() == produced.Load() {
				if processed.Load() != int32(totalTasks) {
					t.Errorf("Expected %d tasks to be processed, got %d",
						totalTasks, processed.Load())
				}
				return
			}
		}
	}
}

func TestElasticPool_WorkerScalingBoundary(t *testing.T) {
	config := PoolConfig{
		QueueSize:     5,
		MinWorkers:    2,
		MaxWorkers:    4,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
		BackoffTime:   5 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)

	var currentWorkers atomic.Int32
	var maxObservedWorkers atomic.Int32
	var minObservedWorkers atomic.Int32
	minObservedWorkers.Store(999)

	pool.SetProcessor(func(task int) {
		workers := pool.workerCount.Load()
		currentWorkers.Store(workers)

		for {
			max := maxObservedWorkers.Load()
			if workers <= max {
				break
			}
			if maxObservedWorkers.CompareAndSwap(max, workers) {
				break
			}
		}

		for {
			min := minObservedWorkers.Load()
			if workers >= min {
				break
			}
			if minObservedWorkers.CompareAndSwap(min, workers) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
	})

	pool.SetProducer(func(submit func(int)) {
		for i := 0; i < 20; i++ {
			submit(i)
			time.Sleep(10 * time.Millisecond)
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	time.Sleep(time.Second)

	if maxObservedWorkers.Load() != config.MaxWorkers {
		t.Errorf("Expected max workers to reach %d, got %d",
			config.MaxWorkers, maxObservedWorkers.Load())
	}

	time.Sleep(config.IdleTimeout * 3)

	finalWorkers := pool.workerCount.Load()
	if finalWorkers != config.MinWorkers {
		t.Errorf("Expected final workers to be %d, got %d",
			config.MinWorkers, finalWorkers)
	}

	if minObservedWorkers.Load() != config.MinWorkers {
		t.Errorf("Expected min workers to reach %d, got %d",
			config.MinWorkers, minObservedWorkers.Load())
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("Failed to close pool: %v", err)
	}
}

func TestElasticPool_PerformanceMetrics(t *testing.T) {
	config := PoolConfig{
		QueueSize:     1000,
		MinWorkers:    2,
		MaxWorkers:    10,
		ScaleInterval: 50 * time.Millisecond,
		IdleTimeout:   200 * time.Millisecond,
	}

	pool := NewElasticPool[int](config)

	var totalProcessingTime atomic.Int64
	var totalTasks atomic.Int32
	startTime := time.Now()

	pool.SetProcessor(func(task int) {
		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		totalProcessingTime.Add(time.Since(start).Nanoseconds())
		totalTasks.Add(1)
	})

	numTasks := 10000
	pool.SetProducer(func(submit func(int)) {
		for i := 0; i < numTasks; i++ {
			submit(i)
		}
	})

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	deadline := time.After(10 * time.Second)
	for totalTasks.Load() < int32(numTasks) {
		select {
		case <-deadline:
			t.Fatal("Test timeout")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	totalTime := time.Since(startTime)
	avgProcessingTime := time.Duration(totalProcessingTime.Load() / int64(totalTasks.Load()))
	throughput := float64(totalTasks.Load()) / totalTime.Seconds()

	t.Logf("Performance Metrics:")
	t.Logf("Total tasks: %d", totalTasks.Load())
	t.Logf("Total time: %v", totalTime)
	t.Logf("Average processing time: %v", avgProcessingTime)
	t.Logf("Throughput: %.2f tasks/second", throughput)
}

func BenchmarkElasticPool(b *testing.B) {
	benchmarks := []struct {
		name       string
		queueSize  int
		minWorkers int32
		maxWorkers int32
		taskTime   time.Duration
	}{
		{"SmallQueue_FastTasks", 100, 2, 5, time.Microsecond},
		{"SmallQueue_SlowTasks", 100, 2, 5, time.Millisecond},
		{"LargeQueue_FastTasks", 1000, 5, 20, time.Microsecond},
		{"LargeQueue_SlowTasks", 1000, 5, 20, time.Millisecond},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			config := PoolConfig{
				QueueSize:     bm.queueSize,
				MinWorkers:    bm.minWorkers,
				MaxWorkers:    bm.maxWorkers,
				ScaleInterval: 50 * time.Millisecond,
				IdleTimeout:   200 * time.Millisecond,
				BackoffTime:   5 * time.Millisecond,
			}

			pool := NewElasticPool[int](config)

			var processed atomic.Int32

			pool.SetProcessor(func(task int) {
				time.Sleep(bm.taskTime)
				processed.Add(1)
			})

			ctx, cancel := context.WithCancel(context.Background())

			pool.SetProducer(func(submit func(int)) {
				i := 0
				for {
					select {
					case <-ctx.Done():
						return
					default:
						submit(i)
						i++
					}
				}
			})

			b.ResetTimer()

			if err := pool.Start(); err != nil {
				b.Fatalf("Failed to start pool: %v", err)
			}

			time.Sleep(time.Second)
			cancel()

			if err := pool.Close(); err != nil {
				b.Fatalf("Failed to shutdown pool: %v", err)
			}

			b.ReportMetric(float64(processed.Load())/float64(time.Second), "tasks/sec")
		})
	}
}

func complexCalculation() float64 {
	result := 0.0
	for i := 0; i < 1000; i++ {
		result += math.Sqrt(float64(i)) * math.Sin(float64(i))
	}
	return result
}

func BenchmarkComplexCalculations(b *testing.B) {
	totalTasks := 1000000
	results := make(map[string]struct {
		duration  time.Duration
		opsPerSec float64
		memStats  runtime.MemStats
	})

	getMemStats := func() runtime.MemStats {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m
	}

	b.Run("SingleThread", func(b *testing.B) {
		start := time.Now()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		for i := 0; i < totalTasks; i++ {
			complexCalculation()
		}

		duration := time.Since(start)
		memAfter := getMemStats()
		results["SingleThread"] = struct {
			duration  time.Duration
			opsPerSec float64
			memStats  runtime.MemStats
		}{
			duration:  duration,
			opsPerSec: float64(totalTasks) / duration.Seconds(),
			memStats:  memAfter,
		}
	})

	threadCounts := []int{4, 6, 8, runtime.NumCPU()}
	for _, threads := range threadCounts {
		name := fmt.Sprintf("FixedThreads_%d", threads)
		b.Run(name, func(b *testing.B) {
			start := time.Now()
			var wg sync.WaitGroup
			taskChan := make(chan int, totalTasks)

			for i := 0; i < threads; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range taskChan {
						complexCalculation()
					}
				}()
			}

			for i := 0; i < totalTasks; i++ {
				taskChan <- i
			}
			close(taskChan)
			wg.Wait()

			duration := time.Since(start)
			memAfter := getMemStats()
			results[name] = struct {
				duration  time.Duration
				opsPerSec float64
				memStats  runtime.MemStats
			}{
				duration:  duration,
				opsPerSec: float64(totalTasks) / duration.Seconds(),
				memStats:  memAfter,
			}
		})
	}

	b.Run("ElasticPool", func(b *testing.B) {
		config := PoolConfig{
			QueueSize:     10000,
			MinWorkers:    int32(3),
			MaxWorkers:    int32(runtime.NumCPU() * 2),
			ScaleInterval: 10 * time.Millisecond,
			IdleTimeout:   10 * time.Millisecond,
			BackoffTime:   10 * time.Millisecond,
		}

		pool := NewElasticPool[int](config)
		start := time.Now()

		pool.SetProcessor(func(task int) {
			complexCalculation()
		})

		pool.SetProducer(func(submit func(int)) {
			for i := 0; i < totalTasks; i++ {
				submit(i)
			}
		})

		if err := pool.Start(); err != nil {
			b.Fatalf("Failed to start pool: %v", err)
		}

		deadline := time.After(5 * time.Minute)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-deadline:
				b.Fatal("Timeout waiting for tasks to complete")
			case <-ticker.C:
				if len(pool.taskQueue) == 0 {
					duration := time.Since(start)
					memAfter := getMemStats()
					results["ElasticPool"] = struct {
						duration  time.Duration
						opsPerSec float64
						memStats  runtime.MemStats
					}{
						duration:  duration,
						opsPerSec: float64(totalTasks) / duration.Seconds(),
						memStats:  memAfter,
					}
					pool.Close()
					goto DONE
				}
			}
		}
	DONE:
	})

	b.Logf("\nPerformance Comparison (Total Tasks: %d):\n", totalTasks)
	b.Logf("%-20s %-15s %-20s %-15s %-15s\n",
		"Method", "Duration", "Ops/Sec", "Memory(MB)", "GC Runs")

	for name, result := range results {
		memUsedMB := float64(result.memStats.Alloc) / 1024 / 1024
		b.Logf("%-20s %-15s %-20.2f %-15.2f %-15d\n",
			name,
			result.duration,
			result.opsPerSec,
			memUsedMB,
			result.memStats.NumGC,
		)
	}
}
