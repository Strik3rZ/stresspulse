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
	"stresspulse/agent"
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
	agentManager  *agent.AgentManager
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
		agentManager:  agent.NewAgentManager(),
	}

	mux := http.NewServeMux()

	staticDir := filepath.Join("web", "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	mux.HandleFunc("/", ws.corsMiddleware(ws.handleIndex))

	mux.HandleFunc("/api/start", ws.corsMiddleware(ws.validateJSONMiddleware(ws.handleStart)))
	mux.HandleFunc("/api/stop", ws.corsMiddleware(ws.handleStop))
	mux.HandleFunc("/api/stats", ws.corsMiddleware(ws.handleStats))
	mux.HandleFunc("/api/logs", ws.corsMiddleware(ws.handleLogs))
	mux.HandleFunc("/api/config", ws.corsMiddleware(ws.handleConfig))

	mux.HandleFunc("/api/agents", ws.corsMiddleware(ws.handleAgents))
	mux.HandleFunc("/api/agents/add", ws.corsMiddleware(ws.validateJSONMiddleware(ws.handleAddAgent)))
	mux.HandleFunc("/api/agents/remove", ws.corsMiddleware(ws.validateJSONMiddleware(ws.handleRemoveAgent)))
	mux.HandleFunc("/api/agents/start", ws.corsMiddleware(ws.validateJSONMiddleware(ws.handleStartAgent)))
	mux.HandleFunc("/api/agents/stop", ws.corsMiddleware(ws.validateJSONMiddleware(ws.handleStopAgent)))
	mux.HandleFunc("/api/agents/stats", ws.corsMiddleware(ws.handleAgentStats))

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return ws
}

func (ws *WebServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (ws *WebServer) validateJSONMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" && contentType != "" {
				http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}
		next(w, r)
	}
}

func (ws *WebServer) Start() error {
	ws.addLog("info", "Web interface starting on port %d", ws.port)
	logger.Info("Web interface available at http://localhost:%d", ws.port)

	ws.ctx, ws.cancel = context.WithCancel(context.Background())

	if err := ws.agentManager.LoadAgents(); err != nil {
		ws.addLog("warning", "Failed to load saved agents: %v", err)
		logger.Warning("Failed to load saved agents: %v", err)
	} else {
		agents := ws.agentManager.GetAgents()
		agentCount := len(agents)
		if agentCount > 0 {
			ws.addLog("info", "Loaded %d saved agent(s)", agentCount)
			logger.Info("Loaded %d saved agent(s)", agentCount)
			
			for agentID, agentInfo := range agents {
				ws.addLog("info", "Agent restored: %s (%s)", agentID, agentInfo.URL)
			}
		}
	}

	go ws.agentManager.StartHealthCheck(ws.ctx, 30*time.Second)

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
		ws.addLog("error", "Invalid JSON in start request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := ws.validateConfig(&config); err != nil {
		ws.addLog("error", "Configuration validation failed: %v", err)
		http.Error(w, fmt.Sprintf("Configuration validation failed: %v", err), http.StatusBadRequest)
		return
	}

	ws.configMutex.Lock()
	ws.config = &config
	ws.configMutex.Unlock()

	ws.addLog("info", "Starting stress tests with web configuration...")

	if ws.cancel != nil {
		ws.cancel()
	}

	ws.stopAllGenerators()

	ws.ctx, ws.cancel = context.WithCancel(context.Background())

	var startErrors []string

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
		if config.HTTP.Headers != nil {
			ws.httpGenerator.SetHeaders(config.HTTP.Headers)
		}
		if config.HTTP.Body != "" {
			ws.httpGenerator.SetBody(config.HTTP.Body)
		}
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
			startErrors = append(startErrors, fmt.Sprintf("gRPC: %v", err))
		} else {
			ws.addLog("success", "gRPC load test started: %s at %d RPS", config.GRPC.Address, config.GRPC.RPS)
		}
	}

	if config.FakeLogsEnabled {
		ws.fakeLogGen = logs.NewFakeLogGenerator(config.FakeLogsType, 1*time.Second, logger.GetLogger())
		ws.fakeLogGen.Start(ws.ctx)
		ws.addLog("success", "Fake logs generator started: type=%s", config.FakeLogsType)
	}

	ws.isRunning = true
	ws.startTime = time.Now()

	response := map[string]interface{}{
		"status": "started",
	}

	if len(startErrors) > 0 {
		response["warnings"] = startErrors
		ws.addLog("warning", "Some generators failed to start: %v", startErrors)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	ws.stopAllGenerators()
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

func (ws *WebServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := ws.agentManager.GetAgents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (ws *WebServer) handleAddAgent(w http.ResponseWriter, r *http.Request) {
	logger.Info("handleAddAgent called - method: %s", r.Method)
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AgentID string `json:"agent_id"`
		URL     string `json:"url"`
	}

	logger.Info("Decoding JSON request body...")
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		logger.Error("Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	logger.Info("Received agent data: ID=%s, URL=%s", data.AgentID, data.URL)
	
	ws.agentManager.AddAgent(data.AgentID, data.URL)
	ws.addLog("info", "Agent added: %s (%s)", data.AgentID, data.URL)

	logger.Info("Sending response...")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
	logger.Info("handleAddAgent completed successfully")
}

func (ws *WebServer) handleRemoveAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ws.agentManager.RemoveAgent(data.AgentID)
	ws.addLog("info", "Agent removed: %s", data.AgentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
}

func (ws *WebServer) handleStartAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AgentID string         `json:"agent_id"`
		Config  agent.AgentConfig `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := ws.agentManager.StartLoad(ws.ctx, data.AgentID, data.Config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ws.addLog("info", "Load started on agent: %s", data.AgentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (ws *WebServer) handleStopAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := ws.agentManager.StopLoad(ws.ctx, data.AgentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ws.addLog("info", "Load stopped on agent: %s", data.AgentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (ws *WebServer) handleAgentStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id parameter is required", http.StatusBadRequest)
		return
	}

	stats, err := ws.agentManager.GetStats(ws.ctx, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
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

func (ws *WebServer) validateConfig(config *WebConfiguration) error {
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
		if config.CPU.Drift < 0 || config.CPU.Drift > 100 {
			return fmt.Errorf("CPU drift must be between 0 and 100")
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
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		valid := false
		for _, method := range validMethods {
			if config.HTTP.Method == method {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid HTTP method: %s", config.HTTP.Method)
		}
		validPatterns := []string{"constant", "spike", "cycle", "ramp", "random"}
		valid = false
		for _, pattern := range validPatterns {
			if config.HTTP.Pattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid HTTP pattern: %s", config.HTTP.Pattern)
		}
	}

	if config.WebSocket.Enabled {
		if config.WebSocket.URL == "" {
			return fmt.Errorf("WebSocket URL cannot be empty")
		}
		if config.WebSocket.CPS <= 0 {
			return fmt.Errorf("WebSocket CPS must be positive")
		}
		if config.WebSocket.MessageInterval <= 0 {
			return fmt.Errorf("WebSocket message interval must be positive")
		}
		if config.WebSocket.MessageSize <= 0 {
			return fmt.Errorf("WebSocket message size must be positive")
		}
		validPatterns := []string{"constant", "spike", "cycle", "ramp", "random"}
		valid := false
		for _, pattern := range validPatterns {
			if config.WebSocket.Pattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid WebSocket pattern: %s", config.WebSocket.Pattern)
		}
	}

	if config.GRPC.Enabled {
		if config.GRPC.Address == "" {
			return fmt.Errorf("gRPC address cannot be empty")
		}
		if config.GRPC.RPS <= 0 {
			return fmt.Errorf("gRPC RPS must be positive")
		}
		validMethods := []string{"health_check", "unary", "server_stream", "client_stream", "bidi_stream"}
		valid := false
		for _, method := range validMethods {
			if config.GRPC.Method == method {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid gRPC method: %s", config.GRPC.Method)
		}
		validPatterns := []string{"constant", "spike", "cycle", "ramp", "random"}
		valid = false
		for _, pattern := range validPatterns {
			if config.GRPC.Pattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid gRPC pattern: %s", config.GRPC.Pattern)
		}
	}

	if config.FakeLogsEnabled {
		validTypes := []string{"java", "web", "microservice", "database", "ecommerce", "generic"}
		valid := false
		for _, logType := range validTypes {
			if config.FakeLogsType == logType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid fake logs type: %s", config.FakeLogsType)
		}
	}

	return nil
}

func (ws *WebServer) stopAllGenerators() {
	if ws.cpuGenerator != nil {
		ws.cpuGenerator.Stop()
		ws.cpuGenerator = nil
	}

	if ws.memGenerator != nil {
		ws.memGenerator.Stop()
		ws.memGenerator = nil
	}

	if ws.httpGenerator != nil {
		ws.httpGenerator.Stop()
		ws.httpGenerator = nil
	}

	if ws.wsGenerator != nil {
		ws.wsGenerator.Stop()
		ws.wsGenerator = nil
	}

	if ws.grpcGenerator != nil {
		ws.grpcGenerator.Stop()
		ws.grpcGenerator = nil
	}

	if ws.fakeLogGen != nil {
		ws.fakeLogGen.Stop()
		ws.fakeLogGen = nil
	}
} 