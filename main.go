package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"stresspulse/config"
	"stresspulse/load"
	"stresspulse/logs"
	"stresspulse/logger"
	"stresspulse/memory"
	"stresspulse/metrics"
	"stresspulse/network"
)

func main() {
	cfg := config.NewConfig()
	cfg.ParseFlags()

	logger.Init(cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		logger.Error("Configuration error: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	generator := load.NewGenerator(cfg)

	var fakeLogGenerator *logs.FakeLogGenerator
	if cfg.FakeLogsEnabled {
		fakeLogGenerator = logs.NewFakeLogGenerator(cfg.FakeLogsType, cfg.FakeLogsInterval, logger.GetLogger())
	}

	var memoryGenerator *memory.MemoryGenerator
	if cfg.MemoryEnabled {
		memoryGenerator = memory.NewMemoryGenerator(cfg.MemoryTargetMB, cfg.MemoryPattern, cfg.MemoryInterval)
	}

	var httpGenerator *network.HTTPGenerator
	if cfg.HTTPEnabled {
		httpGenerator = network.NewHTTPGenerator(cfg.HTTPTargetURL, cfg.HTTPTargetRPS, cfg.HTTPPattern, cfg.HTTPMethod, cfg.HTTPTimeout)
		
		if cfg.HTTPHeaders != "" {
			headers := parseHTTPHeaders(cfg.HTTPHeaders)
			httpGenerator.SetHeaders(headers)
		}
		
		if cfg.HTTPBody != "" {
			httpGenerator.SetBody(cfg.HTTPBody)
		}
	}

	if cfg.MetricsEnabled {
		collector := metrics.NewCollector()
		go func() {
			if err := collector.StartServer(cfg.MetricsPort); err != nil {
				logger.Error("Metrics server error: %v", err)
			}
		}()
		
		if cfg.MemoryEnabled {
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						updateMemoryMetrics(memoryGenerator, cfg.MemoryTargetMB)
					}
				}
			}()
		}
		
		if cfg.HTTPEnabled {
			go func() {
				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()
				
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						updateHTTPMetrics(httpGenerator, cfg.HTTPTargetRPS)
					}
				}
			}()
		}
	}

	generator.Start(ctx)

	if cfg.FakeLogsEnabled {
		fakeLogGenerator.Start()
	}

	if cfg.MemoryEnabled {
		memoryGenerator.Start(ctx)
	}

	if cfg.HTTPEnabled {
		httpGenerator.Start(ctx)
	}

	logger.Info("Starting StressPulse - Advanced Load Generator")
	logger.Info("Target CPU: %.1f%%", cfg.TargetCPUPercent)
	logger.Info("Drift Amplitude: %.1f%%", cfg.DriftAmplitude)
	logger.Info("Drift Period: %s", cfg.DriftPeriod)
	logger.Info("Pattern Type: %s", cfg.PatternType)
	logger.Info("Number of Workers: %d", cfg.NumWorkers)
	if cfg.Duration > 0 {
		logger.Info("Duration: %s", cfg.Duration)
	} else {
		logger.Info("Duration: indefinite")
	}
	if cfg.MetricsEnabled {
		logger.Info("Metrics enabled on port %d", cfg.MetricsPort)
	}
	if cfg.FakeLogsEnabled {
		logger.Info("Fake logs enabled: type=%s, interval=%s", cfg.FakeLogsType, cfg.FakeLogsInterval)
	}
	if cfg.MemoryEnabled {
		logger.Info("Memory stress enabled: target=%dMB, pattern=%s, interval=%s", cfg.MemoryTargetMB, cfg.MemoryPattern, cfg.MemoryInterval)
	}
	if cfg.HTTPEnabled {
		logger.Info("HTTP load test enabled: url=%s, target=%d RPS, pattern=%s, method=%s", cfg.HTTPTargetURL, cfg.HTTPTargetRPS, cfg.HTTPPattern, cfg.HTTPMethod)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if cfg.Duration > 0 {
		select {
		case <-time.After(cfg.Duration):
			logger.Info("Test duration completed")
		case sig := <-sigChan:
			logger.Info("Received signal: %v", sig)
		}
	} else {
		sig := <-sigChan
		logger.Info("Received signal: %v", sig)
	}

	if cfg.FakeLogsEnabled {
		fakeLogGenerator.Stop()
	}

	if cfg.MemoryEnabled {
		memoryGenerator.Stop()
	}

	if cfg.HTTPEnabled {
		httpGenerator.Stop()
	}

	generator.Stop()

	logger.Info("StressPulse Final Statistics:")
	
	stats := generator.GetStats()
	logger.Info("CPU - Average Load: %.1f%%, Runtime: %s, Samples: %d", 
		stats.AverageLoad, time.Since(stats.StartTime), stats.TotalSamples)

	if cfg.MemoryEnabled {
		memStats := memoryGenerator.GetStats()
		logger.Info("Memory - Peak: %dMB, Allocated: %.1fMB, Released: %.1fMB, Operations: %d", 
			memStats.AllocatedMB, 
			float64(memStats.TotalAllocated)/(1024*1024),
			float64(memStats.TotalReleased)/(1024*1024),
			memStats.AllocationCount)
	}

	if cfg.HTTPEnabled {
		httpStats := httpGenerator.GetStats()
		avgResponseTime := httpGenerator.GetAverageResponseTime()
		successRate := httpGenerator.GetSuccessRate()
		
		logger.Info("HTTP - Total: %d, Success: %d, Failed: %d, Avg Response: %s, Success Rate: %.1f%%",
			httpStats.TotalRequests,
			httpStats.SuccessRequests, 
			httpStats.FailedRequests,
			avgResponseTime,
			successRate)
	}
}

func updateMemoryMetrics(memGen *memory.MemoryGenerator, targetMB int) {
	if memGen == nil {
		return
	}

	stats := memGen.GetStats()
	
	metrics.MemoryAllocatedGauge.Set(float64(stats.AllocatedMB))
	metrics.MemoryTargetGauge.Set(float64(targetMB))
	metrics.MemoryTotalAllocatedCounter.Add(float64(stats.TotalAllocated))
	metrics.MemoryTotalReleasedCounter.Add(float64(stats.TotalReleased))
	metrics.MemoryAllocationOpsCounter.Add(float64(stats.AllocationCount))

	allocated, totalAlloc, sys := memory.GetSystemMemoryStats()
	metrics.MemorySystemAllocatedGauge.Set(float64(allocated))
	metrics.MemorySystemTotalGauge.Set(float64(totalAlloc))
	metrics.MemorySystemSysGauge.Set(float64(sys))
}

func updateHTTPMetrics(httpGen *network.HTTPGenerator, targetRPS int) {
	if httpGen == nil {
		return
	}

	stats := httpGen.GetStats()
	avgResponseTime := httpGen.GetAverageResponseTime()
	successRate := httpGen.GetSuccessRate()

	metrics.HTTPRequestsCounter.Add(float64(stats.TotalRequests))
	metrics.HTTPSuccessCounter.Add(float64(stats.SuccessRequests))
	metrics.HTTPFailedCounter.Add(float64(stats.FailedRequests))
	metrics.HTTPCurrentRPSGauge.Set(float64(stats.CurrentRPS))
	metrics.HTTPTargetRPSGauge.Set(float64(targetRPS))
	
	metrics.HTTPResponseTimeGauge.Set(avgResponseTime.Seconds())
	if stats.MinResponseTime < time.Hour {
		metrics.HTTPMinResponseTimeGauge.Set(stats.MinResponseTime.Seconds())
	}
	metrics.HTTPMaxResponseTimeGauge.Set(stats.MaxResponseTime.Seconds())
	
	metrics.HTTPResponseTimeHistogram.Observe(avgResponseTime.Seconds())
	
	metrics.HTTPSuccessRateGauge.Set(successRate)
}

func parseHTTPHeaders(headersStr string) map[string]string {
	headers := make(map[string]string)
	
	if headersStr == "" {
		return headers
	}
	
	pairs := strings.Split(headersStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				headers[key] = value
			}
		}
	}
	
	return headers
} 