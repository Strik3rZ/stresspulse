package network

import (
	"context"
	"crypto/tls"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"stresspulse/logger"
)

type GRPCGenerator struct {
	targetAddress   string
	targetRPS       int
	pattern         string
	serviceName     string
	methodType      string
	useSecure       bool
	enabled         bool
	ctx             context.Context
	cancel          context.CancelFunc
	stats           *GRPCStats
	requestChan     chan struct{}
	workerCount     int
	connPool        []*grpc.ClientConn
	poolSize        int
	metadata        map[string]string
}

type GRPCStats struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	TotalResponseTime time.Duration
	MinResponseTime   time.Duration
	MaxResponseTime   time.Duration
	CurrentRPS        int64
	StartTime         time.Time
	StatusCodes       map[codes.Code]int64
	mutex             sync.RWMutex
}

func NewGRPCGenerator(targetAddress string, targetRPS int, pattern, serviceName, methodType string, useSecure bool) *GRPCGenerator {
	workerCount := 10
	poolSize := 5
	
	if targetRPS > 500 {
		workerCount = targetRPS / 50
		poolSize = targetRPS / 200
		
		if workerCount > 100 {
			workerCount = 100
		}
		if poolSize > 20 {
			poolSize = 20 // gRPC соединения дорогие
		}
		if poolSize < 5 {
			poolSize = 5
		}
	}

	return &GRPCGenerator{
		targetAddress: targetAddress,
		targetRPS:     targetRPS,
		pattern:       pattern,
		serviceName:   serviceName,
		methodType:    methodType,
		useSecure:     useSecure,
		enabled:       false,
		requestChan:   make(chan struct{}, targetRPS*4),
		workerCount:   workerCount,
		poolSize:      poolSize,
		connPool:      make([]*grpc.ClientConn, 0),
		metadata:      make(map[string]string),
		stats: &GRPCStats{
			StartTime:       time.Now(),
			MinResponseTime: time.Hour,
			StatusCodes:     make(map[codes.Code]int64),
		},
	}
}

func (gg *GRPCGenerator) SetMetadata(metadata map[string]string) {
	gg.metadata = metadata
}

func (gg *GRPCGenerator) Start(ctx context.Context) error {
	if gg.enabled {
		return nil
	}

	gg.enabled = true
	gg.ctx, gg.cancel = context.WithCancel(ctx)
	
	if err := gg.createConnectionPool(); err != nil {
		return err
	}
	
	logger.Info("Starting gRPC load generator: %s, target RPS: %d, pattern: %s", 
		gg.targetAddress, gg.targetRPS, gg.pattern)
	
	for i := 0; i < gg.workerCount; i++ {
		go gg.grpcWorker(i)
	}
	
	go gg.generateRPSLoad()
	go gg.statsCollector()
	
	return nil
}

func (gg *GRPCGenerator) Stop() {
	if !gg.enabled {
		return
	}

	gg.enabled = false
	if gg.cancel != nil {
		gg.cancel()
	}
	
	for _, conn := range gg.connPool {
		conn.Close()
	}
	gg.connPool = nil
	
	logger.Info("gRPC load generator stopped")
}

func (gg *GRPCGenerator) createConnectionPool() error {
	var creds credentials.TransportCredentials
	if gg.useSecure {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		creds = insecure.NewCredentials()
	}
	
	for i := 0; i < gg.poolSize; i++ {
		conn, err := grpc.DialContext(gg.ctx, gg.targetAddress, 
			grpc.WithTransportCredentials(creds),
			grpc.WithBlock(),
			grpc.WithTimeout(30*time.Second),
		)
		if err != nil {
			for _, existingConn := range gg.connPool {
				existingConn.Close()
			}
			return err
		}
		gg.connPool = append(gg.connPool, conn)
	}
	
	return nil
}

func (gg *GRPCGenerator) generateRPSLoad() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastSecond := time.Now().Unix()
	requestsThisSecond := 0

	for {
		select {
		case <-gg.ctx.Done():
			return
		case <-ticker.C:
			currentSecond := time.Now().Unix()
			
			if currentSecond != lastSecond {
				lastSecond = currentSecond
				requestsThisSecond = 0
			}
			
			currentRPS := gg.calculateCurrentRPS()
			requestsToSend := gg.calculateRequestsToSend(currentRPS, requestsThisSecond)
			
			for i := 0; i < requestsToSend; i++ {
				select {
				case gg.requestChan <- struct{}{}:
					requestsThisSecond++
				default:
				}
			}
		}
	}
}

func (gg *GRPCGenerator) calculateCurrentRPS() int {
	switch gg.pattern {
	case "constant":
		return gg.targetRPS
	case "spike":
		if rand.Intn(10) == 0 {
			return gg.targetRPS * 3
		}
		return gg.targetRPS
	case "cycle":
		elapsedSeconds := int(time.Since(gg.stats.StartTime).Seconds())
		cyclePosition := (elapsedSeconds / 30) % 4
		
		switch cyclePosition {
		case 0:
			return gg.targetRPS / 4
		case 1:
			return gg.targetRPS
		case 2:
			return gg.targetRPS / 2
		case 3:
			return gg.targetRPS / 8
		}
	case "ramp":
		elapsedMinutes := int(time.Since(gg.stats.StartTime).Minutes())
		rampMultiplier := float64(elapsedMinutes+1) * 0.2
		if rampMultiplier > 1.0 {
			rampMultiplier = 1.0
		}
		return int(float64(gg.targetRPS) * rampMultiplier)
	case "random":
		variation := rand.Intn(140) + 10
		return (gg.targetRPS * variation) / 100
	default:
		return gg.targetRPS
	}
	return gg.targetRPS
}

func (gg *GRPCGenerator) calculateRequestsToSend(currentRPS, requestsThisSecond int) int {
	requestsPer100ms := currentRPS / 10
	
	remainingRPS := currentRPS - requestsThisSecond
	if remainingRPS < 0 {
		remainingRPS = 0
	}
	
	if requestsPer100ms > remainingRPS {
		requestsPer100ms = remainingRPS
	}
	
	return requestsPer100ms
}

func (gg *GRPCGenerator) grpcWorker(workerID int) {
	logger.Debug("gRPC worker %d started", workerID)
	defer logger.Debug("gRPC worker %d stopped", workerID)

	for {
		select {
		case <-gg.ctx.Done():
			return
		case <-gg.requestChan:
			gg.makeRequest(workerID)
		}
	}
}

func (gg *GRPCGenerator) makeRequest(workerID int) {
	startTime := time.Now()
	
	conn := gg.connPool[workerID%len(gg.connPool)]
	
	ctx := gg.ctx
	if len(gg.metadata) > 0 {
		md := metadata.New(gg.metadata)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var err error
	
	switch gg.methodType {
	case "health_check":
		err = gg.makeHealthCheckRequest(ctx, conn)
	case "unary":
		err = gg.makeUnaryRequest(ctx, conn)
	case "server_stream":
		err = gg.makeServerStreamRequest(ctx, conn)
	case "client_stream":
		err = gg.makeClientStreamRequest(ctx, conn)
	case "bidi_stream":
		err = gg.makeBidiStreamRequest(ctx, conn)
	default:
		err = gg.makeHealthCheckRequest(ctx, conn)
	}
	
	responseTime := time.Since(startTime)
	
	if err != nil {
		gg.recordFailure(responseTime, err)
		logger.Debug("gRPC request failed: %v", err)
	} else {
		gg.recordSuccess(responseTime)
	}
}

func (gg *GRPCGenerator) makeHealthCheckRequest(ctx context.Context, conn *grpc.ClientConn) error {
	client := grpc_health_v1.NewHealthClient(conn)
	
	req := &grpc_health_v1.HealthCheckRequest{
		Service: gg.serviceName,
	}
	
	_, err := client.Check(ctx, req)
	return err
}

func (gg *GRPCGenerator) makeUnaryRequest(ctx context.Context, conn *grpc.ClientConn) error {
	return gg.makeHealthCheckRequest(ctx, conn)
}

func (gg *GRPCGenerator) makeServerStreamRequest(ctx context.Context, conn *grpc.ClientConn) error {
	client := grpc_health_v1.NewHealthClient(conn)
	
	req := &grpc_health_v1.HealthCheckRequest{
		Service: gg.serviceName,
	}
	
	stream, err := client.Watch(ctx, req)
	if err != nil {
		return err
	}
	defer stream.CloseSend()
	
	for i := 0; i < 3; i++ {
		_, err := stream.Recv()
		if err != nil {
			break
		}
	}
	
	return nil
}

func (gg *GRPCGenerator) makeClientStreamRequest(ctx context.Context, conn *grpc.ClientConn) error {
	for i := 0; i < 3; i++ {
		if err := gg.makeHealthCheckRequest(ctx, conn); err != nil {
			return err
		}
	}
	return nil
}

func (gg *GRPCGenerator) makeBidiStreamRequest(ctx context.Context, conn *grpc.ClientConn) error {
	client := grpc_health_v1.NewHealthClient(conn)
	
	req := &grpc_health_v1.HealthCheckRequest{
		Service: gg.serviceName,
	}
	
	stream, err := client.Watch(ctx, req)
	if err != nil {
		return err
	}
	defer stream.CloseSend()
	
	_, err = stream.Recv()
	return err
}

func (gg *GRPCGenerator) recordSuccess(responseTime time.Duration) {
	atomic.AddInt64(&gg.stats.TotalRequests, 1)
	atomic.AddInt64(&gg.stats.SuccessRequests, 1)
	
	gg.stats.mutex.Lock()
	gg.stats.TotalResponseTime += responseTime
	if responseTime < gg.stats.MinResponseTime {
		gg.stats.MinResponseTime = responseTime
	}
	if responseTime > gg.stats.MaxResponseTime {
		gg.stats.MaxResponseTime = responseTime
	}
	gg.stats.StatusCodes[codes.OK]++
	gg.stats.mutex.Unlock()
}

func (gg *GRPCGenerator) recordFailure(responseTime time.Duration, err error) {
	atomic.AddInt64(&gg.stats.TotalRequests, 1)
	atomic.AddInt64(&gg.stats.FailedRequests, 1)
	
	gg.stats.mutex.Lock()
	gg.stats.TotalResponseTime += responseTime
	
	if st, ok := status.FromError(err); ok {
		gg.stats.StatusCodes[st.Code()]++
	} else {
		gg.stats.StatusCodes[codes.Unknown]++
	}
	gg.stats.mutex.Unlock()
}

func (gg *GRPCGenerator) statsCollector() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	lastRequests := int64(0)
	
	for {
		select {
		case <-gg.ctx.Done():
			return
		case <-ticker.C:
			currentRequests := atomic.LoadInt64(&gg.stats.TotalRequests)
			currentRPS := currentRequests - lastRequests
			atomic.StoreInt64(&gg.stats.CurrentRPS, currentRPS)
			lastRequests = currentRequests
		}
	}
}

func (gg *GRPCGenerator) GetStats() *GRPCStats {
	gg.stats.mutex.RLock()
	defer gg.stats.mutex.RUnlock()
	
	total := atomic.LoadInt64(&gg.stats.TotalRequests)
	success := atomic.LoadInt64(&gg.stats.SuccessRequests)
	failed := atomic.LoadInt64(&gg.stats.FailedRequests)
	currentRPS := atomic.LoadInt64(&gg.stats.CurrentRPS)
	
	statusCodes := make(map[codes.Code]int64)
	for code, count := range gg.stats.StatusCodes {
		statusCodes[code] = count
	}
	
	return &GRPCStats{
		TotalRequests:     total,
		SuccessRequests:   success,
		FailedRequests:    failed,
		CurrentRPS:        currentRPS,
		TotalResponseTime: gg.stats.TotalResponseTime,
		MinResponseTime:   gg.stats.MinResponseTime,
		MaxResponseTime:   gg.stats.MaxResponseTime,
		StartTime:         gg.stats.StartTime,
		StatusCodes:       statusCodes,
	}
}

func (gg *GRPCGenerator) GetAverageResponseTime() time.Duration {
	stats := gg.GetStats()
	if stats.TotalRequests == 0 {
		return 0
	}
	return stats.TotalResponseTime / time.Duration(stats.TotalRequests)
}

func (gg *GRPCGenerator) GetSuccessRate() float64 {
	stats := gg.GetStats()
	if stats.TotalRequests == 0 {
		return 0
	}
	return float64(stats.SuccessRequests) / float64(stats.TotalRequests) * 100.0
} 