package publisher

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	LastPublishedVersion = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "last_published_version",
	})

	PublishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "publish_errors",
	})
)
