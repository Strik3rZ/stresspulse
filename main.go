package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stresspulse/config"
	"stresspulse/load"
	"stresspulse/logs"
	"stresspulse/logger"
	"stresspulse/metrics"
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

	if cfg.MetricsEnabled {
		collector := metrics.NewCollector()
		go func() {
			if err := collector.StartServer(cfg.MetricsPort); err != nil {
				logger.Error("Metrics server error: %v", err)
			}
		}()
	}

	generator.Start(ctx)

	if cfg.FakeLogsEnabled {
		fakeLogGenerator.Start()
	}

	logger.Info("Starting StressPulse - Advanced CPU Load Generator")
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

	generator.Stop()

	stats := generator.GetStats()
	logger.Info("StressPulse Final Statistics:")
	logger.Info("Average CPU Load: %.1f%%", stats.AverageLoad)
	logger.Info("Total Runtime: %s", time.Since(stats.StartTime))
	logger.Info("Total Samples: %d", stats.TotalSamples)
} 