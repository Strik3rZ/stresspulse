package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CPULoadGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_stress_current_load",
		Help: "Current CPU load percentage",
	})

	CPUAverageLoadGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_stress_average_load",
		Help: "Average CPU load percentage",
	})

	CPULoadSamplesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cpu_stress_samples_total",
		Help: "Total number of CPU load samples collected",
	})

	CPULoadDurationGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_stress_duration_seconds",
		Help: "Duration of the CPU stress test in seconds",
	})

	MemoryAllocatedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_allocated_mb",
		Help: "Currently allocated memory in MB",
	})

	MemoryTargetGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_target_mb",
		Help: "Target memory allocation in MB",
	})

	MemoryTotalAllocatedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "memory_total_allocated_bytes",
		Help: "Total memory allocated during the test",
	})

	MemoryTotalReleasedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "memory_total_released_bytes",
		Help: "Total memory released during the test",
	})

	MemoryAllocationOpsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "memory_allocation_operations_total",
		Help: "Total number of memory allocation operations",
	})

	MemorySystemAllocatedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_system_allocated_bytes",
		Help: "System allocated memory from runtime stats",
	})

	MemorySystemTotalGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_system_total_bytes",
		Help: "Total system memory from runtime stats",
	})

	MemorySystemSysGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_system_sys_bytes",
		Help: "System memory obtained from OS",
	})

	HTTPRequestsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests sent",
	})

	HTTPSuccessCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_requests_success_total",
		Help: "Total number of successful HTTP requests",
	})

	HTTPFailedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_requests_failed_total",
		Help: "Total number of failed HTTP requests",
	})

	HTTPCurrentRPSGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_per_second",
		Help: "Current HTTP requests per second",
	})

	HTTPTargetRPSGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_target_rps",
		Help: "Target HTTP requests per second",
	})

	HTTPResponseTimeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_response_time_seconds",
		Help:    "HTTP response time distribution",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
	})

	HTTPResponseTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_avg_response_time_seconds",
		Help: "Average HTTP response time",
	})

	HTTPMinResponseTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_min_response_time_seconds",
		Help: "Minimum HTTP response time",
	})

	HTTPMaxResponseTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_max_response_time_seconds",
		Help: "Maximum HTTP response time",
	})

	HTTPSuccessRateGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_success_rate_percent",
		Help: "HTTP success rate in percentage",
	})

	WebSocketConnectionsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_connections_total",
		Help: "Total number of WebSocket connections attempted",
	})

	WebSocketActiveConnectionsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_active_connections",
		Help: "Current number of active WebSocket connections",
	})

	WebSocketFailedConnectionsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_connections_failed_total",
		Help: "Total number of failed WebSocket connections",
	})

	WebSocketCurrentCPSGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_connections_per_second",
		Help: "Current WebSocket connections per second",
	})

	WebSocketTargetCPSGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_target_cps",
		Help: "Target WebSocket connections per second",
	})

	WebSocketMessagesSentCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_messages_sent_total",
		Help: "Total number of WebSocket messages sent",
	})

	WebSocketMessagesReceivedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_messages_received_total",
		Help: "Total number of WebSocket messages received",
	})

	WebSocketConnectionTimeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "websocket_connection_time_seconds",
		Help:    "WebSocket connection time distribution",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
	})

	WebSocketAvgConnectionTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_avg_connection_time_seconds",
		Help: "Average WebSocket connection time",
	})

	WebSocketSuccessRateGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_success_rate_percent",
		Help: "WebSocket connection success rate in percentage",
	})

	GRPCRequestsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total number of gRPC requests sent",
	})

	GRPCSuccessCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grpc_requests_success_total",
		Help: "Total number of successful gRPC requests",
	})

	GRPCFailedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "grpc_requests_failed_total",
		Help: "Total number of failed gRPC requests",
	})

	GRPCCurrentRPSGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grpc_requests_per_second",
		Help: "Current gRPC requests per second",
	})

	GRPCTargetRPSGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grpc_target_rps",
		Help: "Target gRPC requests per second",
	})

	GRPCResponseTimeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "grpc_response_time_seconds",
		Help:    "gRPC response time distribution",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
	})

	GRPCResponseTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grpc_avg_response_time_seconds",
		Help: "Average gRPC response time",
	})

	GRPCMinResponseTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grpc_min_response_time_seconds",
		Help: "Minimum gRPC response time",
	})

	GRPCMaxResponseTimeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grpc_max_response_time_seconds",
		Help: "Maximum gRPC response time",
	})

	GRPCSuccessRateGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grpc_success_rate_percent",
		Help: "gRPC success rate in percentage",
	})

	GRPCStatusCodesCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_status_codes_total",
		Help: "Total number of gRPC requests by status code",
	}, []string{"code"})
) 