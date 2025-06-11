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
) 