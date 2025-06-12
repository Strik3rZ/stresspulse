package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"stresspulse/logger"
)

type HTTPGenerator struct {
	targetURL        string
	targetRPS        int
	pattern          string
	method           string
	headers          map[string]string
	body             string
	timeout          time.Duration
	enabled          bool
	client           *http.Client
	ctx              context.Context
	cancel           context.CancelFunc
	stats            *HTTPStats
	requestChan      chan struct{}
	workerCount      int
}

type HTTPStats struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalResponseTime time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	CurrentRPS        int64
	StartTime         time.Time
	mutex             sync.RWMutex
}

type RequestTemplate struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func NewHTTPGenerator(targetURL string, targetRPS int, pattern, method string, timeout time.Duration) *HTTPGenerator {
	workerCount := 10
	if targetRPS > 1000 {
		workerCount = targetRPS / 100
		if workerCount > 100 {
			workerCount = 100
		}
	}

	return &HTTPGenerator{
		targetURL:   targetURL,
		targetRPS:   targetRPS,
		pattern:     pattern,
		method:      method,
		headers:     make(map[string]string),
		timeout:     timeout,
		enabled:     false,
		requestChan: make(chan struct{}, targetRPS*4),
		workerCount: workerCount,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        200,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
			},
		},
		stats: &HTTPStats{
			StartTime:       time.Now(),
			MinResponseTime: time.Hour,
		},
	}
}

func (hg *HTTPGenerator) SetHeaders(headers map[string]string) {
	hg.headers = headers
}

func (hg *HTTPGenerator) SetBody(body string) {
	hg.body = body
}

func (hg *HTTPGenerator) Start(ctx context.Context) {
	if hg.enabled {
		return
	}

	hg.enabled = true
	hg.ctx = ctx
	
	logger.Info("Starting HTTP load generator: %s, target RPS: %d, pattern: %s", 
		hg.targetURL, hg.targetRPS, hg.pattern)
	
	for i := 0; i < hg.workerCount; i++ {
		go hg.httpWorker(i)
	}
	
	go hg.generateRPSLoad()
	
	go hg.statsCollector()
}

func (hg *HTTPGenerator) Stop() {
	if !hg.enabled {
		return
	}

	hg.enabled = false
	
	logger.Info("HTTP load generator stopped")
}

func (hg *HTTPGenerator) generateRPSLoad() {
	tickInterval := 100 * time.Millisecond
	if hg.targetRPS > 5000 {
		tickInterval = 50 * time.Millisecond
	}
	if hg.targetRPS > 10000 {
		tickInterval = 20 * time.Millisecond
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	lastSecond := time.Now().Unix()
	requestsThisSecond := 0

	for {
		select {
		case <-hg.ctx.Done():
			return
		case <-ticker.C:
			currentSecond := time.Now().Unix()
			
			if currentSecond != lastSecond {
				lastSecond = currentSecond
				requestsThisSecond = 0
			}
			
			currentRPS := hg.calculateCurrentRPS()
			requestsToSend := hg.calculateRequestsToSend(currentRPS, requestsThisSecond, tickInterval)
			
			for i := 0; i < requestsToSend; i++ {
				select {
				case hg.requestChan <- struct{}{}:
					requestsThisSecond++
				default:
				}
			}
		}
	}
}

func (hg *HTTPGenerator) calculateCurrentRPS() int {
	switch hg.pattern {
	case "constant":
		return hg.targetRPS
	case "spike":
		if rand.Intn(10) == 0 {
			return hg.targetRPS * 3
		}
		return hg.targetRPS
	case "cycle":
		elapsedSeconds := int(time.Since(hg.stats.StartTime).Seconds())
		cyclePosition := (elapsedSeconds / 30) % 4
		
		switch cyclePosition {
		case 0:
			return hg.targetRPS / 4
		case 1:
			return hg.targetRPS
		case 2:
			return hg.targetRPS / 2
		case 3:
			return hg.targetRPS / 8
		}
	case "ramp":
		elapsedMinutes := int(time.Since(hg.stats.StartTime).Minutes())
		rampMultiplier := float64(elapsedMinutes+1) * 0.2
		if rampMultiplier > 1.0 {
			rampMultiplier = 1.0
		}
		return int(float64(hg.targetRPS) * rampMultiplier)
	case "random":
		variation := rand.Intn(140) + 10
		return (hg.targetRPS * variation) / 100
	default:
		return hg.targetRPS
	}
	return hg.targetRPS
}

func (hg *HTTPGenerator) calculateRequestsToSend(currentRPS, requestsThisSecond int, tickInterval time.Duration) int {
	ticksPerSecond := float64(time.Second) / float64(tickInterval)
	requestsPerTick := float64(currentRPS) / ticksPerSecond
	
	remainingRPS := currentRPS - requestsThisSecond
	if remainingRPS < 0 {
		remainingRPS = 0
	}
	
	requestsToSend := int(requestsPerTick)
	if requestsToSend > remainingRPS {
		requestsToSend = remainingRPS
	}
	
	return requestsToSend
}

func (hg *HTTPGenerator) httpWorker(workerID int) {
	logger.Debug("HTTP worker %d started", workerID)
	defer logger.Debug("HTTP worker %d stopped", workerID)

	for {
		select {
		case <-hg.ctx.Done():
			return
		case <-hg.requestChan:
			hg.makeRequest()
		}
	}
}

func (hg *HTTPGenerator) makeRequest() {
	startTime := time.Now()
	
	var bodyReader io.Reader
	if hg.body != "" {
		bodyReader = bytes.NewReader([]byte(hg.body))
	}
	
	req, err := http.NewRequestWithContext(hg.ctx, hg.method, hg.targetURL, bodyReader)
	if err != nil {
		hg.recordFailure(time.Since(startTime))
		logger.Debug("Failed to create request: %v", err)
		return
	}
	
	for key, value := range hg.headers {
		req.Header.Set(key, value)
	}
	
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "StressPulse/1.0")
	}
	
	resp, err := hg.client.Do(req)
	responseTime := time.Since(startTime)
	
	if err != nil {
		hg.recordFailure(responseTime)
		logger.Debug("Request failed: %v", err)
		return
	}
	
	defer resp.Body.Close()
	
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		logger.Debug("Failed to read response body: %v", err)
	}
	
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		hg.recordSuccess(responseTime)
	} else {
		hg.recordFailure(responseTime)
		logger.Debug("Request failed with status: %d", resp.StatusCode)
	}
}

func (hg *HTTPGenerator) recordSuccess(responseTime time.Duration) {
	atomic.AddInt64(&hg.stats.TotalRequests, 1)
	atomic.AddInt64(&hg.stats.SuccessRequests, 1)
	
	hg.stats.mutex.Lock()
	hg.stats.TotalResponseTime += responseTime
	if responseTime < hg.stats.MinResponseTime {
		hg.stats.MinResponseTime = responseTime
	}
	if responseTime > hg.stats.MaxResponseTime {
		hg.stats.MaxResponseTime = responseTime
	}
	hg.stats.mutex.Unlock()
}

func (hg *HTTPGenerator) recordFailure(responseTime time.Duration) {
	atomic.AddInt64(&hg.stats.TotalRequests, 1)
	atomic.AddInt64(&hg.stats.FailedRequests, 1)
	
	hg.stats.mutex.Lock()
	hg.stats.TotalResponseTime += responseTime
	hg.stats.mutex.Unlock()
}

func (hg *HTTPGenerator) statsCollector() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	lastRequests := int64(0)
	
	for {
		select {
		case <-hg.ctx.Done():
			return
		case <-ticker.C:
			currentRequests := atomic.LoadInt64(&hg.stats.TotalRequests)
			currentRPS := currentRequests - lastRequests
			atomic.StoreInt64(&hg.stats.CurrentRPS, currentRPS)
			lastRequests = currentRequests
		}
	}
}

func (hg *HTTPGenerator) GetStats() *HTTPStats {
	hg.stats.mutex.RLock()
	defer hg.stats.mutex.RUnlock()
	
	total := atomic.LoadInt64(&hg.stats.TotalRequests)
	success := atomic.LoadInt64(&hg.stats.SuccessRequests)
	failed := atomic.LoadInt64(&hg.stats.FailedRequests)
	currentRPS := atomic.LoadInt64(&hg.stats.CurrentRPS)
	
	return &HTTPStats{
		TotalRequests:     total,
		SuccessRequests:   success,
		FailedRequests:    failed,
		CurrentRPS:        currentRPS,
		TotalResponseTime: hg.stats.TotalResponseTime,
		MinResponseTime:   hg.stats.MinResponseTime,
		MaxResponseTime:   hg.stats.MaxResponseTime,
		StartTime:         hg.stats.StartTime,
	}
}

func (hg *HTTPGenerator) GetAverageResponseTime() time.Duration {
	stats := hg.GetStats()
	if stats.TotalRequests == 0 {
		return 0
	}
	return stats.TotalResponseTime / time.Duration(stats.TotalRequests)
}

func (hg *HTTPGenerator) GetSuccessRate() float64 {
	stats := hg.GetStats()
	if stats.TotalRequests == 0 {
		return 0
	}
	return float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100.0
}

func CreateRequestTemplate(method string, headers map[string]string, bodyData interface{}) (*RequestTemplate, error) {
	var body string
	
	if bodyData != nil {
		if bodyBytes, err := json.Marshal(bodyData); err == nil {
			body = string(bodyBytes)
		} else {
			return nil, fmt.Errorf("failed to marshal body: %v", err)
		}
	}
	
	return &RequestTemplate{
		Method:  method,
		Headers: headers,
		Body:    body,
	}, nil
} 