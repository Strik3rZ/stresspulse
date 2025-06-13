package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"stresspulse/config"
	"stresspulse/load"
	"stresspulse/logger"
	"stresspulse/memory"
	"stresspulse/network"
	"stresspulse/logs"
)

type Agent struct {
	server        *http.Server
	port          int
	cpuGenerator  *load.Generator
	memGenerator  *memory.MemoryGenerator
	httpGenerator *network.HTTPGenerator
	wsGenerator   *network.WebSocketGenerator
	grpcGenerator *network.GRPCGenerator
	fakeLogGen    *logs.FakeLogGenerator
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	startTime     time.Time
}

type AgentConfig struct {
	CPU struct {
		Enabled bool    `json:"enabled"`
		Load    float64 `json:"load"`
		Pattern string  `json:"pattern"`
		Drift   float64 `json:"drift"`
	} `json:"cpu"`
	Memory struct {
		Enabled bool   `json:"enabled"`
		Target  int    `json:"target"`
		Pattern string `json:"pattern"`
	} `json:"memory"`
	HTTP struct {
		Enabled bool              `json:"enabled"`
		URL     string            `json:"url"`
		RPS     int               `json:"rps"`
		Method  string            `json:"method"`
		Pattern string            `json:"pattern"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	} `json:"http"`
	WebSocket struct {
		Enabled         bool   `json:"enabled"`
		URL             string `json:"url"`
		CPS             int    `json:"cps"`
		Pattern         string `json:"pattern"`
		MessageInterval int    `json:"messageInterval"`
		MessageSize     int    `json:"messageSize"`
	} `json:"websocket"`
	GRPC struct {
		Enabled bool   `json:"enabled"`
		Address string `json:"address"`
		RPS     int    `json:"rps"`
		Method  string `json:"method"`
		Pattern string `json:"pattern"`
		Secure  bool   `json:"secure"`
		Service string `json:"service"`
	} `json:"grpc"`
	FakeLogsEnabled bool   `json:"fakeLogsEnabled"`
	FakeLogsType    string `json:"fakeLogsType"`
}

func NewAgent(port int) *Agent {
	return &Agent{
		port:      port,
		startTime: time.Now(),
	}
}

func (a *Agent) Start() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/start", a.handleStart)
	mux.HandleFunc("/api/stop", a.handleStop)
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/api/health", a.handleHealth)

	a.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Agent server error: %v", err)
		}
	}()

	logger.Info("Agent started on port %d", a.port)
	return nil
}

func (a *Agent) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("agent server shutdown error: %v", err)
	}

	a.stopAllGenerators()
	logger.Info("Agent stopped")
	return nil
}

func (a *Agent) validateAgentConfig(config *AgentConfig) error {
	if config.CPU.Enabled {
		if config.CPU.Load < 0 || config.CPU.Load > 100 {
			return fmt.Errorf("CPU load must be between 0 and 100")
		}
		validPatterns := []string{"sine", "square", "sawtooth", "random"}
		valid := false
		for _, pattern := range validPatterns {
			if config.CPU.Pattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid CPU pattern: %s", config.CPU.Pattern)
		}
	}

	if config.Memory.Enabled {
		if config.Memory.Target <= 0 {
			return fmt.Errorf("memory target must be positive")
		}
		validPatterns := []string{"constant", "leak", "spike", "cycle", "random"}
		valid := false
		for _, pattern := range validPatterns {
			if config.Memory.Pattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid memory pattern: %s", config.Memory.Pattern)
		}
	}

	if config.HTTP.Enabled {
		if config.HTTP.URL == "" {
			return fmt.Errorf("HTTP URL cannot be empty")
		}
		if config.HTTP.RPS <= 0 {
			return fmt.Errorf("HTTP RPS must be positive")
		}
	}

	if config.WebSocket.Enabled {
		if config.WebSocket.URL == "" {
			return fmt.Errorf("WebSocket URL cannot be empty")
		}
		if config.WebSocket.CPS <= 0 {
			return fmt.Errorf("WebSocket CPS must be positive")
		}
	}

	if config.GRPC.Enabled {
		if config.GRPC.Address == "" {
			return fmt.Errorf("gRPC address cannot be empty")
		}
		if config.GRPC.RPS <= 0 {
			return fmt.Errorf("gRPC RPS must be positive")
		}
	}

	return nil
}

func (a *Agent) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var agentConfig AgentConfig
	if err := json.NewDecoder(r.Body).Decode(&agentConfig); err != nil {
		logger.Error("Agent: Invalid JSON in start request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := a.validateAgentConfig(&agentConfig); err != nil {
		logger.Error("Agent: Configuration validation failed: %v", err)
		http.Error(w, fmt.Sprintf("Configuration validation failed: %v", err), http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.stopAllGenerators()

	var startErrors []string

	if agentConfig.CPU.Enabled {
		cpuConfig := &config.Config{
			TargetCPUPercent: agentConfig.CPU.Load,
			PatternType:      agentConfig.CPU.Pattern,
			DriftAmplitude:   agentConfig.CPU.Drift,
		}
		a.cpuGenerator = load.NewGenerator(cpuConfig)
		a.cpuGenerator.Start(a.ctx)
		logger.Info("Agent: CPU stress test started: %.1f%% load with %s pattern", agentConfig.CPU.Load, agentConfig.CPU.Pattern)
	}

	if agentConfig.Memory.Enabled {
		a.memGenerator = memory.NewMemoryGenerator(agentConfig.Memory.Target, agentConfig.Memory.Pattern, 2*time.Second)
		a.memGenerator.Start(a.ctx)
		logger.Info("Agent: Memory stress test started: %d MB with %s pattern", agentConfig.Memory.Target, agentConfig.Memory.Pattern)
	}

	if agentConfig.HTTP.Enabled {
		a.httpGenerator = network.NewHTTPGenerator(agentConfig.HTTP.URL, agentConfig.HTTP.RPS, agentConfig.HTTP.Pattern, agentConfig.HTTP.Method, 10*time.Second)
		if agentConfig.HTTP.Headers != nil {
			a.httpGenerator.SetHeaders(agentConfig.HTTP.Headers)
		}
		if agentConfig.HTTP.Body != "" {
			a.httpGenerator.SetBody(agentConfig.HTTP.Body)
		}
		a.httpGenerator.Start(a.ctx)
		logger.Info("Agent: HTTP load test started: %s %s at %d RPS", agentConfig.HTTP.Method, agentConfig.HTTP.URL, agentConfig.HTTP.RPS)
	}

	if agentConfig.WebSocket.Enabled {
		a.wsGenerator = network.NewWebSocketGenerator(
			agentConfig.WebSocket.URL,
			agentConfig.WebSocket.CPS,
			agentConfig.WebSocket.Pattern,
			time.Duration(agentConfig.WebSocket.MessageInterval)*time.Second,
			agentConfig.WebSocket.MessageSize,
		)
		a.wsGenerator.Start(a.ctx)
		logger.Info("Agent: WebSocket load test started: %s at %d CPS", agentConfig.WebSocket.URL, agentConfig.WebSocket.CPS)
	}

	if agentConfig.GRPC.Enabled {
		a.grpcGenerator = network.NewGRPCGenerator(
			agentConfig.GRPC.Address,
			agentConfig.GRPC.RPS,
			agentConfig.GRPC.Pattern,
			agentConfig.GRPC.Service,
			agentConfig.GRPC.Method,
			agentConfig.GRPC.Secure,
		)
		if err := a.grpcGenerator.Start(a.ctx); err != nil {
			logger.Error("Agent: Failed to start gRPC generator: %v", err)
			startErrors = append(startErrors, fmt.Sprintf("gRPC: %v", err))
		} else {
			logger.Info("Agent: gRPC load test started: %s at %d RPS", agentConfig.GRPC.Address, agentConfig.GRPC.RPS)
		}
	}

	if agentConfig.FakeLogsEnabled {
		a.fakeLogGen = logs.NewFakeLogGenerator(agentConfig.FakeLogsType, 1*time.Second, logger.GetLogger())
		a.fakeLogGen.Start(a.ctx)
		logger.Info("Agent: Fake logs generator started: type=%s", agentConfig.FakeLogsType)
	}

	response := map[string]interface{}{
		"status": "started",
	}

	if len(startErrors) > 0 {
		response["warnings"] = startErrors
		logger.Warning("Agent: Some generators failed to start: %v", startErrors)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *Agent) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.stopAllGenerators()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (a *Agent) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := make(map[string]interface{})

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	uptime := time.Since(a.startTime)
	
	stats["agent_status"] = "healthy"
	stats["uptime"] = uptime.String()
	stats["system"] = map[string]interface{}{
		"goroutines":     runtime.NumGoroutine(),
		"memory_alloc":   memStats.Alloc / 1024 / 1024, // MB
		"memory_sys":     memStats.Sys / 1024 / 1024,   // MB
		"memory_heap":    memStats.HeapAlloc / 1024 / 1024, // MB
		"gc_runs":        memStats.NumGC,
		"cpu_cores":      runtime.NumCPU(),
	}

	if a.cpuGenerator != nil {
		cpuStats := a.cpuGenerator.GetStats()
		stats["cpu"] = map[string]interface{}{
			"CurrentLoad":  cpuStats.CurrentLoad,
			"AverageLoad":  cpuStats.AverageLoad,
			"TotalSamples": cpuStats.TotalSamples,
			"StartTime":    cpuStats.StartTime,
			"LastUpdate":   cpuStats.LastUpdate,
		}
	}

	if a.memGenerator != nil {
		memGenStats := a.memGenerator.GetStats()
		stats["memory"] = map[string]interface{}{
			"AllocatedMB":     memGenStats.AllocatedMB,
			"TotalAllocated":  memGenStats.TotalAllocated,
			"TotalReleased":   memGenStats.TotalReleased,
			"AllocationCount": memGenStats.AllocationCount,
			"StartTime":       memGenStats.StartTime,
		}
	}

	if a.httpGenerator != nil {
		httpStats := a.httpGenerator.GetStats()
		stats["http"] = map[string]interface{}{
			"CurrentRPS":        httpStats.CurrentRPS,
			"TotalRequests":     httpStats.TotalRequests,
			"SuccessRequests":   httpStats.SuccessRequests,
			"FailedRequests":    httpStats.FailedRequests,
			"SuccessRate":       a.httpGenerator.GetSuccessRate(),
			"AverageResponseTime": a.httpGenerator.GetAverageResponseTime().String(),
			"StartTime":         httpStats.StartTime,
		}
	}

	if a.wsGenerator != nil {
		wsStats := a.wsGenerator.GetStats()
		stats["websocket"] = map[string]interface{}{
			"CurrentCPS":        wsStats.CurrentCPS,
			"ActiveConnections": wsStats.ActiveConnections,
			"TotalConnections":  wsStats.TotalConnections,
			"SuccessRate":       a.wsGenerator.GetSuccessRate(),
			"StartTime":         wsStats.StartTime,
		}
	}

	if a.grpcGenerator != nil {
		grpcStats := a.grpcGenerator.GetStats()
		stats["grpc"] = map[string]interface{}{
			"CurrentRPS":    grpcStats.CurrentRPS,
			"TotalRequests": grpcStats.TotalRequests,
			"SuccessRate":   a.grpcGenerator.GetSuccessRate(),
			"StartTime":     grpcStats.StartTime,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (a *Agent) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (a *Agent) stopAllGenerators() {
	if a.cpuGenerator != nil {
		a.cpuGenerator.Stop()
		a.cpuGenerator = nil
	}

	if a.memGenerator != nil {
		a.memGenerator.Stop()
		a.memGenerator = nil
	}

	if a.httpGenerator != nil {
		a.httpGenerator.Stop()
		a.httpGenerator = nil
	}

	if a.wsGenerator != nil {
		a.wsGenerator.Stop()
		a.wsGenerator = nil
	}

	if a.grpcGenerator != nil {
		a.grpcGenerator.Stop()
		a.grpcGenerator = nil
	}

	if a.fakeLogGen != nil {
		a.fakeLogGen.Stop()
		a.fakeLogGen = nil
	}
} 