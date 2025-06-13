package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"stresspulse/logger"
)

type AgentManager struct {
	agents     map[string]*AgentInfo
	mu         sync.RWMutex
	httpClient *http.Client
	configFile string
}

type AgentInfo struct {
	URL       string    `json:"url"`
	LastSeen  time.Time `json:"last_seen"`
	IsHealthy bool      `json:"is_healthy"`
}

type AgentStats struct {
	AgentID string                 `json:"agent_id"`
	Stats   map[string]interface{} `json:"stats"`
}

type AgentsData struct {
	Agents map[string]*AgentInfo `json:"agents"`
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents: make(map[string]*AgentInfo),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		configFile: "agents.json",
	}
}

func (am *AgentManager) LoadAgents() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	configDir := "config"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, am.configFile)
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read agents config file: %v", err)
	}

	var agentsData AgentsData
	if err := json.Unmarshal(data, &agentsData); err != nil {
		return fmt.Errorf("failed to parse agents config: %v", err)
	}

	for agentID, agentInfo := range agentsData.Agents {
		am.agents[agentID] = agentInfo
		am.agents[agentID].IsHealthy = false
	}

	return nil
}

func (am *AgentManager) SaveAgents() error {
	currentDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get current working directory: %v", err)
	} else {
		logger.Info("Current working directory: %s", currentDir)
	}

	configDir := "config"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("Failed to create config directory: %v", err)
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, am.configFile)
	absPath, _ := filepath.Abs(configPath)
	logger.Info("Attempting to save agents to: %s (absolute: %s)", configPath, absPath)

	am.mu.RLock()
	agentsCopy := make(map[string]*AgentInfo)
	for id, info := range am.agents {
		agentsCopy[id] = &AgentInfo{
			URL:       info.URL,
			LastSeen:  info.LastSeen,
			IsHealthy: info.IsHealthy,
		}
	}
	am.mu.RUnlock()

	agentsData := AgentsData{
		Agents: agentsCopy,
	}

	data, err := json.MarshalIndent(agentsData, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal agents data: %v", err)
		return fmt.Errorf("failed to marshal agents data: %v", err)
	}

	logger.Info("Saving %d agents to config file", len(agentsCopy))
	logger.Info("JSON data to be written: %s", string(data))
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logger.Error("Failed to write agents config file: %v", err)
		return fmt.Errorf("failed to write agents config file: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		logger.Error("File was not created successfully: %v", err)
		return fmt.Errorf("file verification failed: %v", err)
	}

	logger.Info("Successfully saved agents config to: %s", configPath)
	return nil
}

func (am *AgentManager) AddAgent(agentID, url string) {
	logger.Info("Adding agent: %s (%s)", agentID, url)
	
	am.mu.Lock()
	am.agents[agentID] = &AgentInfo{
		URL:       url,
		LastSeen:  time.Now(),
		IsHealthy: false,
	}
	am.mu.Unlock()

	if err := am.SaveAgents(); err != nil {
		logger.Warning("Failed to save agents config: %v", err)
	}
}

func (am *AgentManager) RemoveAgent(agentID string) {
	logger.Info("Removing agent: %s", agentID)
	
	am.mu.Lock()
	delete(am.agents, agentID)
	am.mu.Unlock()

	if err := am.SaveAgents(); err != nil {
		logger.Warning("Failed to save agents config: %v", err)
	}
}

func (am *AgentManager) GetAgents() map[string]*AgentInfo {
	am.mu.RLock()
	defer am.mu.RUnlock()

	agents := make(map[string]*AgentInfo)
	for id, info := range am.agents {
		agents[id] = info
	}
	return agents
}

func (am *AgentManager) StartLoad(ctx context.Context, agentID string, config interface{}) error {
	am.mu.RLock()
	agent, exists := am.agents[agentID]
	am.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", agent.URL+"/api/start", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned status code %d", resp.StatusCode)
	}

	return nil
}

func (am *AgentManager) StopLoad(ctx context.Context, agentID string) error {
	am.mu.RLock()
	agent, exists := am.agents[agentID]
	am.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", agent.URL+"/api/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned status code %d", resp.StatusCode)
	}

	return nil
}

func (am *AgentManager) GetStats(ctx context.Context, agentID string) (*AgentStats, error) {
	am.mu.RLock()
	agent, exists := am.agents[agentID]
	am.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", agent.URL+"/api/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status code %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &AgentStats{
		AgentID: agentID,
		Stats:   stats,
	}, nil
}

func (am *AgentManager) CheckHealth(ctx context.Context) {
	am.mu.Lock()
	defer am.mu.Unlock()

	logger.Info("Starting health check for %d agents", len(am.agents))

	for agentID, agent := range am.agents {
		logger.Info("Checking health for agent %s at %s", agentID, agent.URL)
		
		req, err := http.NewRequestWithContext(ctx, "GET", agent.URL+"/api/health", nil)
		if err != nil {
			logger.Error("Failed to create health check request for agent %s: %v", agentID, err)
			agent.IsHealthy = false
			continue
		}

		resp, err := am.httpClient.Do(req)
		if err != nil {
			logger.Error("Health check failed for agent %s: %v", agentID, err)
			agent.IsHealthy = false
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			logger.Info("Agent %s is healthy (status: %d)", agentID, resp.StatusCode)
			agent.IsHealthy = true
		} else {
			logger.Warning("Agent %s returned unhealthy status: %d", agentID, resp.StatusCode)
			agent.IsHealthy = false
		}
		
		agent.LastSeen = time.Now()
		logger.Info("Agent %s last seen updated to: %s", agentID, agent.LastSeen.Format("15:04:05"))
	}
	
	logger.Info("Health check completed")
}

func (am *AgentManager) StartHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.CheckHealth(ctx)
			if err := am.SaveAgents(); err != nil {
				logger.Warning("Failed to save agents state: %v", err)
			}
		}
	}
} 