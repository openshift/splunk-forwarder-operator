package metrics

import "github.com/prometheus/client_golang/prometheus"

// metricsConfig allows user to specify how to send information to the prometheus instance.
type metricsConfig struct {
	metricsPath        string
	metricsPort        string
	metricsRegisterer  prometheus.Registerer
	metricsGatherer    prometheus.Gatherer
	serviceName        string
	serviceLabel       map[string]string
	collectorList      []prometheus.Collector
	withRoute          bool
	withServiceMonitor bool
	namespace          string
}
