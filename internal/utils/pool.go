package utils

import (
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

type TaskProducer[T any] func(func(T))

type WorkerFunc[T any] func(T)

type ElasticPool[T any] struct {
	taskQueue   chan T
	done        chan struct{}
	workerCount *atomic.Int32
	producer    TaskProducer[T]
	processor   WorkerFunc[T]
	config      PoolConfig
	lastScale   atomic.Value
}

type PoolConfig struct {
	QueueSize     int
	MinWorkers    int32
	MaxWorkers    int32
	ScaleInterval time.Duration
	IdleTimeout   time.Duration
	BackoffTime   time.Duration
}

func NewElasticPool[T any](config PoolConfig) *ElasticPool[T] {
	if config.QueueSize == 0 {
		config.QueueSize = 1024
	}
	if config.MinWorkers == 0 {
		config.MinWorkers = 1
	}
	if config.MaxWorkers == 0 {
		config.MaxWorkers = int32(runtime.NumCPU())
	}
	if config.ScaleInterval == 0 {
		config.ScaleInterval = 100 * time.Millisecond
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 10 * time.Second
	}
	if config.BackoffTime == 0 {
		config.BackoffTime = 10 * time.Millisecond
	}

	p := &ElasticPool[T]{
		taskQueue:   make(chan T, config.QueueSize),
		done:        make(chan struct{}),
		workerCount: &atomic.Int32{},
		config:      config,
	}
	p.lastScale.Store(time.Now())
	return p
}

func (p *ElasticPool[T]) SetProducer(producer TaskProducer[T]) {
	p.producer = producer
}

func (p *ElasticPool[T]) SetProcessor(processor WorkerFunc[T]) {
	p.processor = processor
}

func (p *ElasticPool[T]) Start() error {
	if p.producer == nil {
		return fmt.Errorf("producer not registered")
	}
	if p.processor == nil {
		return fmt.Errorf("processor not registered")
	}
	for i := int32(0); i < p.config.MinWorkers; i++ {
		go p.startWorker()
	}
	go p.startProducer()
	go p.startMonitor()
	return nil
}

func (p *ElasticPool[T]) startProducer() {

	p.producer(func(data T) {
		select {
		case <-p.done:
			return
		case p.taskQueue <- data:
			return
		}
	})
}

func (p *ElasticPool[T]) startWorker() {
	p.workerCount.Add(1)
	defer p.workerCount.Add(-1)

	idleTimeout := time.NewTimer(p.config.IdleTimeout)
	defer idleTimeout.Stop()

	for {
		select {
		case <-p.done:
			return
		case data, ok := <-p.taskQueue:
			if !ok {
				return
			}

			if !idleTimeout.Stop() {
				select {
				case <-idleTimeout.C:
				default:
				}
			}
			idleTimeout.Reset(p.config.IdleTimeout)
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Worker recovered from panic: %v", r)
					}
				}()
				p.processor(data)
			}()
		case <-idleTimeout.C:
			if p.workerCount.Load() > p.config.MinWorkers {
				return
			}
			idleTimeout.Reset(p.config.IdleTimeout)
		}
	}
}

func (p *ElasticPool[T]) startMonitor() {
	ticker := time.NewTicker(p.config.ScaleInterval)
	defer ticker.Stop()

	var queueLen int
	var currentWorkers int32

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			queueLen = len(p.taskQueue)
			if queueLen > p.config.QueueSize/2 {
				currentWorkers = p.workerCount.Load()
				if currentWorkers < p.config.MaxWorkers {
					go p.startWorker()
				}
			}
		}
	}
}

func (p *ElasticPool[T]) Close() error {
	close(p.done)
	close(p.taskQueue)
	return nil
}
