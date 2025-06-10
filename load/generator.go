package load

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"stresspulse/config"
	"stresspulse/logger"
	"stresspulse/metrics"
	"stresspulse/patterns"
)

type Generator struct {
	config  *config.Config
	wg      sync.WaitGroup
	done    chan struct{}
	stats   *Stats
	pattern patterns.Pattern
}

type Stats struct {
	mu            sync.RWMutex
	CurrentLoad   float64
	AverageLoad   float64
	TotalSamples  int64
	StartTime     time.Time
	LastUpdate    time.Time
	LoadHistory   []float64
}

func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		config:  cfg,
		done:    make(chan struct{}),
		pattern: patterns.NewPattern(cfg.PatternType),
		stats: &Stats{
			StartTime:   time.Now(),
			LastUpdate:  time.Now(),
			LoadHistory: make([]float64, 0),
		},
	}
}

func (g *Generator) Start(ctx context.Context) {
	workUnits := int(float64(runtime.NumCPU()) * g.config.TargetCPUPercent / 100.0)
	if workUnits < 1 {
		workUnits = 1
	}

	logger.Info("Starting CPU stress test with %d workers", workUnits)
	logger.Debug("Configuration: %+v", g.config)

	g.wg.Add(workUnits)
	for i := 0; i < workUnits; i++ {
		go g.worker(ctx, i)
	}

	if g.config.MetricsEnabled {
		logger.Info("Metrics collection enabled on port %d", g.config.MetricsPort)
		go g.collectStats(ctx)
	}
}

func (g *Generator) Stop() {
	logger.Info("Stopping CPU stress test")
	close(g.done)
	g.wg.Wait()

	if g.config.SaveProfile {
		g.saveProfile()
	}
}

func (g *Generator) GetStats() *Stats {
	g.stats.mu.RLock()
	defer g.stats.mu.RUnlock()
	return g.stats
}

func (g *Generator) worker(ctx context.Context, workerID int) {
	defer g.wg.Done()
	startTime := time.Now()
	logger.Debug("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Worker %d stopped by context", workerID)
			return
		case <-g.done:
			logger.Debug("Worker %d stopped by done signal", workerID)
			return
		default:
			elapsed := time.Since(startTime)
			currentLoad := g.pattern.GetLoad(elapsed, g.config.TargetCPUPercent, g.config.DriftAmplitude)

			if currentLoad < 0 {
				currentLoad = 0
				logger.Warning("Worker %d: Load adjusted from negative to 0", workerID)
			}
			if currentLoad > 100 {
				currentLoad = 100
				logger.Warning("Worker %d: Load adjusted from >100 to 100", workerID)
			}

			workDuration := time.Duration(float64(time.Second) * currentLoad / 100.0)
			restDuration := time.Second - workDuration

			g.updateStats(currentLoad)

			start := time.Now()
			for time.Since(start) < workDuration {
				_ = math.Sqrt(float64(time.Now().UnixNano()))
			}

			time.Sleep(restDuration)
		}
	}
}

func (g *Generator) updateStats(currentLoad float64) {
	g.stats.mu.Lock()
	defer g.stats.mu.Unlock()

	g.stats.CurrentLoad = currentLoad
	g.stats.TotalSamples++
	g.stats.AverageLoad = (g.stats.AverageLoad*float64(g.stats.TotalSamples-1) + currentLoad) / float64(g.stats.TotalSamples)
	g.stats.LastUpdate = time.Now()
	
	if len(g.stats.LoadHistory) >= 1000 {
		g.stats.LoadHistory = g.stats.LoadHistory[1:]
	}
	g.stats.LoadHistory = append(g.stats.LoadHistory, currentLoad)

	if g.config.MetricsEnabled {
		metrics.CPULoadGauge.Set(currentLoad)
		metrics.CPUAverageLoadGauge.Set(g.stats.AverageLoad)
		metrics.CPULoadSamplesCounter.Inc()
		metrics.CPULoadDurationGauge.Set(time.Since(g.stats.StartTime).Seconds())
	}
}

func (g *Generator) collectStats(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-g.done:
			return
		case <-ticker.C:
			stats := g.GetStats()
			logger.Debug("Current stats: load=%.2f%%, avg=%.2f%%, samples=%d",
				stats.CurrentLoad, stats.AverageLoad, stats.TotalSamples)
		}
	}
}

func (g *Generator) saveProfile() {
	profile := struct {
		Config     *config.Config
		Stats      *Stats
		Timestamp  time.Time
		Duration   time.Duration
	}{
		Config:    g.config,
		Stats:     g.stats,
		Timestamp: time.Now(),
		Duration:  time.Since(g.stats.StartTime),
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal profile: %v", err)
		return
	}

	if err := os.WriteFile(g.config.ProfilePath, data, 0644); err != nil {
		logger.Error("Failed to save profile to %s: %v", g.config.ProfilePath, err)
		return
	}

	logger.Info("Profile saved to %s", g.config.ProfilePath)
} 