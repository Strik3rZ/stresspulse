package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	cfg "stresspulse/config"
	"stresspulse/load"
	"stresspulse/logger"
	"stresspulse/memory"
	"stresspulse/network"
	"stresspulse/logs"
)

type WebServer struct {
	server        *http.Server
	port          int
	cpuGenerator  *load.Generator
	memGenerator  *memory.MemoryGenerator
	httpGenerator *network.HTTPGenerator
	wsGenerator   *network.WebSocketGenerator
	grpcGenerator *network.GRPCGenerator
	fakeLogGen    *logs.FakeLogGenerator
	logBuffer     []LogEntry
	logMutex      sync.RWMutex
	maxLogEntries int
	startTime     time.Time
	isRunning     bool
	config        *WebConfiguration
	configMutex   sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type WebConfiguration struct {
	CPU struct {
		Enabled bool   `json:"enabled"`
		Load    int    `json:"load"`
		Pattern string `json:"pattern"`
		Drift   int    `json:"drift"`
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
	Duration        string `json:"duration"`
	Workers         int    `json:"workers"`
}

type StatsResponse struct {
	CPU struct {
		Enabled bool    `json:"enabled"`
		Current float64 `json:"current"`
		Target  int     `json:"target"`
	} `json:"cpu,omitempty"`
	Memory struct {
		Enabled bool    `json:"enabled"`
		Current float64 `json:"current"`
		Target  int     `json:"target"`
	} `json:"memory,omitempty"`
	HTTP struct {
		Enabled     bool    `json:"enabled"`
		CurrentRPS  int64   `json:"currentRPS"`
		TargetRPS   int     `json:"targetRPS"`
		SuccessRate float64 `json:"successRate"`
	} `json:"http,omitempty"`
	WebSocket struct {
		Enabled           bool    `json:"enabled"`
		CurrentCPS        int64   `json:"currentCPS"`
		ActiveConnections int64   `json:"activeConnections"`
		SuccessRate       float64 `json:"successRate"`
	} `json:"websocket,omitempty"`
	GRPC struct {
		Enabled     bool    `json:"enabled"`
		CurrentRPS  int64   `json:"currentRPS"`
		TargetRPS   int     `json:"targetRPS"`
		SuccessRate float64 `json:"successRate"`
	} `json:"grpc,omitempty"`
}

func NewWebServer(port int) *WebServer {
	ws := &WebServer{
		port:          port,
		maxLogEntries: 1000,
		logBuffer:     make([]LogEntry, 0),
		config:        &WebConfiguration{},
	}

	mux := http.NewServeMux()

	staticDir := filepath.Join("web", "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	mux.HandleFunc("/", ws.handleIndex)

	mux.HandleFunc("/api/start", ws.handleStart)
	mux.HandleFunc("/api/stop", ws.handleStop)
	mux.HandleFunc("/api/stats", ws.handleStats)
	mux.HandleFunc("/api/logs", ws.handleLogs)
	mux.HandleFunc("/api/config", ws.handleConfig)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return ws
}

func (ws *WebServer) Start() error {
	ws.addLog("info", "Web interface starting on port %d", ws.port)
	logger.Info("Web interface available at http://localhost:%d", ws.port)

	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Web server error: %v", err)
		}
	}()

	return nil
}

func (ws *WebServer) Stop() error {
	ws.addLog("info", "Web interface stopping...")
	
	if ws.cancel != nil {
		ws.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ws.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("web server shutdown error: %v", err)
	}

	ws.addLog("info", "Web interface stopped")
	return nil
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	indexPath := filepath.Join("web", "index.html")
	http.ServeFile(w, r, indexPath)
}

func (ws *WebServer) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config WebConfiguration
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ws.configMutex.Lock()
	ws.config = &config
	ws.configMutex.Unlock()

	ws.addLog("info", "Starting stress tests with web configuration...")

	ws.ctx, ws.cancel = context.WithCancel(context.Background())

	if config.CPU.Enabled {
		cpuConfig := &cfg.Config{
			TargetCPUPercent: float64(config.CPU.Load),
			PatternType:      config.CPU.Pattern,
			DriftAmplitude:   float64(config.CPU.Drift),
		}
		ws.cpuGenerator = load.NewGenerator(cpuConfig)
		ws.cpuGenerator.Start(ws.ctx)
		ws.addLog("success", "CPU stress test started: %d%% load with %s pattern", config.CPU.Load, config.CPU.Pattern)
	}

	if config.Memory.Enabled {
		ws.memGenerator = memory.NewMemoryGenerator(config.Memory.Target, config.Memory.Pattern, 2*time.Second)
		ws.memGenerator.Start(ws.ctx)
		ws.addLog("success", "Memory stress test started: %d MB with %s pattern", config.Memory.Target, config.Memory.Pattern)
	}

	if config.HTTP.Enabled {
		ws.httpGenerator = network.NewHTTPGenerator(config.HTTP.URL, config.HTTP.RPS, config.HTTP.Pattern, config.HTTP.Method, 10*time.Second)
		ws.httpGenerator.SetHeaders(config.HTTP.Headers)
		ws.httpGenerator.SetBody(config.HTTP.Body)
		ws.httpGenerator.Start(ws.ctx)
		ws.addLog("success", "HTTP load test started: %s %s at %d RPS", config.HTTP.Method, config.HTTP.URL, config.HTTP.RPS)
	}

	if config.WebSocket.Enabled {
		ws.wsGenerator = network.NewWebSocketGenerator(
			config.WebSocket.URL,
			config.WebSocket.CPS,
			config.WebSocket.Pattern,
			time.Duration(config.WebSocket.MessageInterval)*time.Second,
			config.WebSocket.MessageSize,
		)
		ws.wsGenerator.Start(ws.ctx)
		ws.addLog("success", "WebSocket load test started: %s at %d CPS", config.WebSocket.URL, config.WebSocket.CPS)
	}

	if config.GRPC.Enabled {
		ws.grpcGenerator = network.NewGRPCGenerator(
			config.GRPC.Address,
			config.GRPC.RPS,
			config.GRPC.Pattern,
			config.GRPC.Service,
			config.GRPC.Method,
			config.GRPC.Secure,
		)
		if err := ws.grpcGenerator.Start(ws.ctx); err != nil {
			ws.addLog("error", "Failed to start gRPC generator: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ws.addLog("success", "gRPC load test started: %s at %d RPS", config.GRPC.Address, config.GRPC.RPS)
	}

	if config.FakeLogsEnabled {
		ws.fakeLogGen = logs.NewFakeLogGenerator(config.FakeLogsType, 1*time.Second, logger.GetLogger())
		ws.fakeLogGen.Start(ws.ctx)
		ws.addLog("success", "Fake logs generator started: type=%s", config.FakeLogsType)
	}

	ws.isRunning = true
	ws.startTime = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (ws *WebServer) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.addLog("info", "Stopping all stress tests...")

	if ws.cancel != nil {
		ws.cancel()
	}

	if ws.cpuGenerator != nil {
		ws.cpuGenerator.Stop()
		ws.cpuGenerator = nil
		ws.addLog("info", "CPU stress test stopped")
	}

	if ws.memGenerator != nil {
		ws.memGenerator.Stop()
		ws.memGenerator = nil
		ws.addLog("info", "Memory stress test stopped")
	}

	if ws.httpGenerator != nil {
		ws.httpGenerator.Stop()
		ws.httpGenerator = nil
		ws.addLog("info", "HTTP load test stopped")
	}

	if ws.wsGenerator != nil {
		ws.wsGenerator.Stop()
		ws.wsGenerator = nil
		ws.addLog("info", "WebSocket load test stopped")
	}

	if ws.grpcGenerator != nil {
		ws.grpcGenerator.Stop()
		ws.grpcGenerator = nil
		ws.addLog("info", "gRPC load test stopped")
	}

	if ws.fakeLogGen != nil {
		ws.fakeLogGen.Stop()
		ws.fakeLogGen = nil
		ws.addLog("info", "Fake logs generator stopped")
	}

	ws.isRunning = false

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (ws *WebServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := StatsResponse{}

	ws.configMutex.RLock()
	config := ws.config
	ws.configMutex.RUnlock()

	if ws.cpuGenerator != nil && config.CPU.Enabled {
		cpuStats := ws.cpuGenerator.GetStats()
		stats.CPU.Enabled = true
		stats.CPU.Current = cpuStats.CurrentLoad
		stats.CPU.Target = config.CPU.Load
	}

	if ws.memGenerator != nil && config.Memory.Enabled {
		memStats := ws.memGenerator.GetStats()
		stats.Memory.Enabled = true
		stats.Memory.Current = float64(memStats.AllocatedMB)
		stats.Memory.Target = config.Memory.Target
	}

	if ws.httpGenerator != nil && config.HTTP.Enabled {
		httpStats := ws.httpGenerator.GetStats()
		stats.HTTP.Enabled = true
		stats.HTTP.CurrentRPS = httpStats.CurrentRPS
		stats.HTTP.TargetRPS = config.HTTP.RPS
		stats.HTTP.SuccessRate = ws.httpGenerator.GetSuccessRate()
	}

	if ws.wsGenerator != nil && config.WebSocket.Enabled {
		wsStats := ws.wsGenerator.GetStats()
		stats.WebSocket.Enabled = true
		stats.WebSocket.CurrentCPS = wsStats.CurrentCPS
		stats.WebSocket.ActiveConnections = wsStats.ActiveConnections
		stats.WebSocket.SuccessRate = ws.wsGenerator.GetSuccessRate()
	}

	if ws.grpcGenerator != nil && config.GRPC.Enabled {
		grpcStats := ws.grpcGenerator.GetStats()
		stats.GRPC.Enabled = true
		stats.GRPC.CurrentRPS = grpcStats.CurrentRPS
		stats.GRPC.TargetRPS = config.GRPC.RPS
		stats.GRPC.SuccessRate = ws.grpcGenerator.GetSuccessRate()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (ws *WebServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	ws.logMutex.RLock()
	logs := make([]LogEntry, 0)
	start := len(ws.logBuffer) - limit
	if start < 0 {
		start = 0
	}
	for i := start; i < len(ws.logBuffer); i++ {
		logs = append(logs, ws.logBuffer[i])
	}
	ws.logMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (ws *WebServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	ws.configMutex.RLock()
	config := ws.config
	ws.configMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (ws *WebServer) addLog(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	
	ws.logMutex.Lock()
	ws.logBuffer = append(ws.logBuffer, LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})

	if len(ws.logBuffer) > ws.maxLogEntries {
		ws.logBuffer = ws.logBuffer[len(ws.logBuffer)-ws.maxLogEntries:]
	}
	ws.logMutex.Unlock()
} 