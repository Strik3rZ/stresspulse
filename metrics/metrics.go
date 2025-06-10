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
) 