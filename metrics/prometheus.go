package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	cpuLoadGauge     prometheus.Gauge
	cpuLoadHistogram prometheus.Histogram
	startTime        time.Time
}

func NewCollector() *Collector {
	return &Collector{
		cpuLoadGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "cpu_load_percentage",
			Help: "Current CPU load percentage",
		}),
		cpuLoadHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "cpu_load_distribution",
			Help:    "Distribution of CPU load over time",
			Buckets: prometheus.LinearBuckets(0, 10, 11), // 0, 10, 20, ..., 100
		}),
		startTime: time.Now(),
	}
}

func (c *Collector) StartServer(port int) error {
	prometheus.MustRegister(c.cpuLoadGauge)
	prometheus.MustRegister(c.cpuLoadHistogram)

	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+string(port), nil)
}

func (c *Collector) UpdateMetrics(currentLoad float64) {
	c.cpuLoadGauge.Set(currentLoad)
	c.cpuLoadHistogram.Observe(currentLoad)
} 