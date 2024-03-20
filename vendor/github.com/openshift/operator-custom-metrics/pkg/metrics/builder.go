package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// Default variables for metrics-path and metrics-port.
const (
	defaultMetricsPath = "/custommetrics"
	defaultMetricsPort = "8089"
)

// metricsConfigBuilder builds a new metricsConfig object.
type metricsConfigBuilder struct {
	config metricsConfig
}

// NewBuilder initializes the configuration builder for a given namespace and service name
func NewBuilder(namespace, serviceName string) *metricsConfigBuilder {
	return &metricsConfigBuilder{
		config: metricsConfig{
			namespace:     namespace,
			metricsPath:   defaultMetricsPath,
			metricsPort:   defaultMetricsPort,
			serviceName:   serviceName,
			collectorList: nil,
		},
	}
}

// GetConfig returns the reference to the built metricsConfig.
func (b *metricsConfigBuilder) GetConfig() *metricsConfig {
	return &b.config
}

// WithPort updates the metrics port to the value provided by the user.
func (b *metricsConfigBuilder) WithPort(port string) *metricsConfigBuilder {
	b.config.metricsPort = port
	return b
}

// WithPort updates the metrics port to the value provided by the user.
func (b *metricsConfigBuilder) WithServiceLabel(label map[string]string) *metricsConfigBuilder {
	b.config.serviceLabel = label
	return b
}

// WithPath updates the metrics path to the value provided by the user.
func (b *metricsConfigBuilder) WithPath(path string) *metricsConfigBuilder {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	b.config.metricsPath = path
	return b
}

// WithCollector appends the prometheus-collector provided by the user to a list of Collectors.
func (b *metricsConfigBuilder) WithCollector(collector prometheus.Collector) *metricsConfigBuilder {
	if b.config.collectorList == nil {
		b.config.collectorList = make([]prometheus.Collector, 0)
	}
	b.config.collectorList = append(b.config.collectorList, collector)
	return b
}

// WithCollectors updates the collectorList to the list of collectors provided by the user.
func (b *metricsConfigBuilder) WithCollectors(collectors []prometheus.Collector) *metricsConfigBuilder {
	b.config.collectorList = collectors
	return b
}

// WithRegistry allows specifying the prometheus registry to use for metrics. Other the default prometheus registry is used.
func (b *metricsConfigBuilder) WithRegistry(registry *prometheus.Registry) *metricsConfigBuilder {
	b.config.metricsRegisterer = registry
	b.config.metricsGatherer = registry
	return b
}

func (b *metricsConfigBuilder) WithRoute() *metricsConfigBuilder {
	b.config.withRoute = true
	return b
}

func (b *metricsConfigBuilder) WithServiceMonitor() *metricsConfigBuilder {
	b.config.withServiceMonitor = true
	return b
}
