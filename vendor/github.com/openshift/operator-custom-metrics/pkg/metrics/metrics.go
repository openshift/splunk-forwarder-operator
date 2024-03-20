// Copyright 2019 RedHat
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartMetrics starts the server based on the metricsConfig provided by the user.
func StartMetrics(config metricsConfig) error {
	if config.metricsRegisterer == nil {
		config.metricsRegisterer = prometheus.DefaultRegisterer
		config.metricsGatherer = prometheus.DefaultGatherer
	}
	err := RegisterMetrics(config.metricsRegisterer, config.collectorList)
	if err != nil {
		return err
	}
	metricsHandler := promhttp.InstrumentMetricHandler(
		config.metricsRegisterer, promhttp.HandlerFor(config.metricsGatherer, promhttp.HandlerOpts{}),
	)
	log.Info(fmt.Sprintf("Port: %s", config.metricsPort))
	metricsPort := fmt.Sprintf(":%s", config.metricsPort)
	if free := isPortFree(metricsPort); !free {
		return fmt.Errorf("port %s is not free", config.metricsPort)
	}
	server := &http.Server{
		Addr:    metricsPort,
		Handler: metricsHandler,
	}
	go server.ListenAndServe() // nolint:errcheck
	return nil
}

func isPortFree(port string) bool {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// RegisterMetrics takes the list of metrics to be registered from the user and
// registers to prometheus.
func RegisterMetrics(metricsRegisterer prometheus.Registerer, list []prometheus.Collector) error {
	for _, metric := range list {
		err := metricsRegisterer.Register(metric)
		if err != nil {
			return err
		}
	}
	return nil
}
