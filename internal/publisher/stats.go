package publisher

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	LastPublishedVersion = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "last_published_version",
	})

	ProvisionedVersions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "provisioned_versions",
	})

	Ticks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ticks",
	})

	TickLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "tick_latency",
		Buckets: []float64{1, 5, 10, 50, 100, 500, 1000, 5000},
	})

	PublishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "publish_errors",
	})
)
