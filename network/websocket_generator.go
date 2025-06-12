package network

import (
	"context"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"stresspulse/logger"
)

type WebSocketGenerator struct {
	targetURL        string
	targetCPS        int
	pattern          string
	messageInterval  time.Duration
	messageSize      int
	enabled          bool
	ctx              context.Context
	cancel           context.CancelFunc
	stats            *WebSocketStats
	connectionChan   chan struct{}
	workerCount      int
	headers          http.Header
	dialer           *websocket.Dialer
}

type WebSocketStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	FailedConnections   int64
	MessagesSent        int64
	MessagesReceived    int64
	TotalResponseTime   time.Duration
	MinResponseTime     time.Duration
	MaxResponseTime     time.Duration
	CurrentCPS          int64
	StartTime           time.Time
	mutex               sync.RWMutex
}

func NewWebSocketGenerator(targetURL string, targetCPS int, pattern string, messageInterval time.Duration, messageSize int) *WebSocketGenerator {
	return &WebSocketGenerator{
		targetURL:       targetURL,
		targetCPS:       targetCPS,
		pattern:         pattern,
		messageInterval: messageInterval,
		messageSize:     messageSize,
		enabled:         false,
		connectionChan:  make(chan struct{}, targetCPS*2),
		workerCount:     10,
		headers:         http.Header{},
		dialer: &websocket.Dialer{
			HandshakeTimeout: 30 * time.Second,
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
		},
		stats: &WebSocketStats{
			StartTime:       time.Now(),
			MinResponseTime: time.Hour,
		},
	}
}

func (wsg *WebSocketGenerator) SetHeaders(headers map[string]string) {
	wsg.headers = http.Header{}
	for key, value := range headers {
		wsg.headers.Set(key, value)
	}
}

func (wsg *WebSocketGenerator) Start(ctx context.Context) {
	if wsg.enabled {
		return
	}

	wsg.enabled = true
	wsg.ctx, wsg.cancel = context.WithCancel(ctx)
	
	logger.Info("Starting WebSocket load generator: %s, target CPS: %d, pattern: %s", 
		wsg.targetURL, wsg.targetCPS, wsg.pattern)
	
	for i := 0; i < wsg.workerCount; i++ {
		go wsg.websocketWorker(i)
	}
	
	go wsg.generateConnectionLoad()
	go wsg.statsCollector()
}

func (wsg *WebSocketGenerator) Stop() {
	if !wsg.enabled {
		return
	}

	wsg.enabled = false
	if wsg.cancel != nil {
		wsg.cancel()
	}
	
	logger.Info("WebSocket load generator stopped")
}

func (wsg *WebSocketGenerator) generateConnectionLoad() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastSecond := time.Now().Unix()
	connectionsThisSecond := 0

	for {
		select {
		case <-wsg.ctx.Done():
			return
		case <-ticker.C:
			currentSecond := time.Now().Unix()
			
			if currentSecond != lastSecond {
				lastSecond = currentSecond
				connectionsThisSecond = 0
			}
			
			currentCPS := wsg.calculateCurrentCPS()
			connectionsToCreate := wsg.calculateConnectionsToCreate(currentCPS, connectionsThisSecond)
			
			for i := 0; i < connectionsToCreate; i++ {
				select {
				case wsg.connectionChan <- struct{}{}:
					connectionsThisSecond++
				default:
				}
			}
		}
	}
}

func (wsg *WebSocketGenerator) calculateCurrentCPS() int {
	switch wsg.pattern {
	case "constant":
		return wsg.targetCPS
	case "spike":
		if rand.Intn(10) == 0 {
			return wsg.targetCPS * 3
		}
		return wsg.targetCPS
	case "cycle":
		elapsedSeconds := int(time.Since(wsg.stats.StartTime).Seconds())
		cyclePosition := (elapsedSeconds / 30) % 4
		
		switch cyclePosition {
		case 0:
			return wsg.targetCPS / 4
		case 1:
			return wsg.targetCPS
		case 2:
			return wsg.targetCPS / 2
		case 3:
			return wsg.targetCPS / 8
		}
	case "ramp":
		elapsedMinutes := int(time.Since(wsg.stats.StartTime).Minutes())
		rampMultiplier := float64(elapsedMinutes+1) * 0.2
		if rampMultiplier > 1.0 {
			rampMultiplier = 1.0
		}
		return int(float64(wsg.targetCPS) * rampMultiplier)
	case "random":
		variation := rand.Intn(140) + 10
		return (wsg.targetCPS * variation) / 100
	default:
		return wsg.targetCPS
	}
	return wsg.targetCPS
}

func (wsg *WebSocketGenerator) calculateConnectionsToCreate(currentCPS, connectionsThisSecond int) int {
	connectionsPer100ms := currentCPS / 10
	
	remainingCPS := currentCPS - connectionsThisSecond
	if remainingCPS < 0 {
		remainingCPS = 0
	}
	
	if connectionsPer100ms > remainingCPS {
		connectionsPer100ms = remainingCPS
	}
	
	return connectionsPer100ms
}

func (wsg *WebSocketGenerator) websocketWorker(workerID int) {
	logger.Debug("WebSocket worker %d started", workerID)
	defer logger.Debug("WebSocket worker %d stopped", workerID)

	for {
		select {
		case <-wsg.ctx.Done():
			return
		case <-wsg.connectionChan:
			wsg.createConnection()
		}
	}
}

func (wsg *WebSocketGenerator) createConnection() {
	startTime := time.Now()
	
	u, err := url.Parse(wsg.targetURL)
	if err != nil {
		wsg.recordFailure(time.Since(startTime))
		logger.Debug("Failed to parse WebSocket URL: %v", err)
		return
	}
	
	conn, resp, err := wsg.dialer.DialContext(wsg.ctx, u.String(), wsg.headers)
	connectionTime := time.Since(startTime)
	
	if err != nil {
		wsg.recordFailure(connectionTime)
		logger.Debug("WebSocket connection failed: %v", err)
		if resp != nil {
			logger.Debug("Response status: %d", resp.StatusCode)
		}
		return
	}
	
	wsg.recordSuccess(connectionTime)
	atomic.AddInt64(&wsg.stats.ActiveConnections, 1)
	
	go wsg.handleConnection(conn)
}

func (wsg *WebSocketGenerator) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&wsg.stats.ActiveConnections, -1)
	}()
	
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	connectionDuration := wsg.getConnectionDuration()
	connectionCtx, connectionCancel := context.WithTimeout(wsg.ctx, connectionDuration)
	defer connectionCancel()
	
	messageData := make([]byte, wsg.messageSize)
	for i := range messageData {
		messageData[i] = byte('A' + (i % 26))
	}
	
	messageTicker := time.NewTicker(wsg.messageInterval)
	defer messageTicker.Stop()
	
	go func() {
		for {
			select {
			case <-connectionCtx.Done():
				return
			default:
				_, _, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						logger.Debug("WebSocket read error: %v", err)
					}
					return
				}
				atomic.AddInt64(&wsg.stats.MessagesReceived, 1)
			}
		}
	}()
	
	for {
		select {
		case <-connectionCtx.Done():
			return
		case <-messageTicker.C:
			err := conn.WriteMessage(websocket.TextMessage, messageData)
			if err != nil {
				logger.Debug("WebSocket write error: %v", err)
				return
			}
			atomic.AddInt64(&wsg.stats.MessagesSent, 1)
			
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		}
	}
}

func (wsg *WebSocketGenerator) getConnectionDuration() time.Duration {
	switch wsg.pattern {
	case "constant":
		return 30*time.Second + time.Duration(rand.Intn(30))*time.Second
	case "spike":
		return 5*time.Second + time.Duration(rand.Intn(10))*time.Second
	case "cycle":
		return 20*time.Second + time.Duration(rand.Intn(40))*time.Second
	case "ramp":
		return 45*time.Second + time.Duration(rand.Intn(30))*time.Second
	case "random":
		return time.Duration(rand.Intn(60)+10) * time.Second
	default:
		return 30 * time.Second
	}
}

func (wsg *WebSocketGenerator) recordSuccess(responseTime time.Duration) {
	atomic.AddInt64(&wsg.stats.TotalConnections, 1)
	
	wsg.stats.mutex.Lock()
	wsg.stats.TotalResponseTime += responseTime
	if responseTime < wsg.stats.MinResponseTime {
		wsg.stats.MinResponseTime = responseTime
	}
	if responseTime > wsg.stats.MaxResponseTime {
		wsg.stats.MaxResponseTime = responseTime
	}
	wsg.stats.mutex.Unlock()
}

func (wsg *WebSocketGenerator) recordFailure(responseTime time.Duration) {
	atomic.AddInt64(&wsg.stats.TotalConnections, 1)
	atomic.AddInt64(&wsg.stats.FailedConnections, 1)
	
	wsg.stats.mutex.Lock()
	wsg.stats.TotalResponseTime += responseTime
	wsg.stats.mutex.Unlock()
}

func (wsg *WebSocketGenerator) statsCollector() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	lastConnections := int64(0)
	
	for {
		select {
		case <-wsg.ctx.Done():
			return
		case <-ticker.C:
			currentConnections := atomic.LoadInt64(&wsg.stats.TotalConnections)
			currentCPS := currentConnections - lastConnections
			atomic.StoreInt64(&wsg.stats.CurrentCPS, currentCPS)
			lastConnections = currentConnections
		}
	}
}

func (wsg *WebSocketGenerator) GetStats() *WebSocketStats {
	wsg.stats.mutex.RLock()
	defer wsg.stats.mutex.RUnlock()
	
	total := atomic.LoadInt64(&wsg.stats.TotalConnections)
	active := atomic.LoadInt64(&wsg.stats.ActiveConnections)
	failed := atomic.LoadInt64(&wsg.stats.FailedConnections)
	messagesSent := atomic.LoadInt64(&wsg.stats.MessagesSent)
	messagesReceived := atomic.LoadInt64(&wsg.stats.MessagesReceived)
	currentCPS := atomic.LoadInt64(&wsg.stats.CurrentCPS)
	
	return &WebSocketStats{
		TotalConnections:  total,
		ActiveConnections: active,
		FailedConnections: failed,
		MessagesSent:      messagesSent,
		MessagesReceived:  messagesReceived,
		CurrentCPS:        currentCPS,
		TotalResponseTime: wsg.stats.TotalResponseTime,
		MinResponseTime:   wsg.stats.MinResponseTime,
		MaxResponseTime:   wsg.stats.MaxResponseTime,
		StartTime:         wsg.stats.StartTime,
	}
}

func (wsg *WebSocketGenerator) GetAverageResponseTime() time.Duration {
	stats := wsg.GetStats()
	if stats.TotalConnections == 0 {
		return 0
	}
	return stats.TotalResponseTime / time.Duration(stats.TotalConnections)
}

func (wsg *WebSocketGenerator) GetSuccessRate() float64 {
	stats := wsg.GetStats()
	if stats.TotalConnections == 0 {
		return 0
	}
	successful := stats.TotalConnections - stats.FailedConnections
	return float64(successful) / float64(stats.TotalConnections) * 100.0
} 