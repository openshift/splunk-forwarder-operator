package main

import (
	"testing"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func TestMetricsPortFromAddr(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		wantPort    int32
		wantEnabled bool
		wantErr     bool
	}{
		{
			name:        "explicit port",
			addr:        ":9999",
			wantPort:    9999,
			wantEnabled: true,
		},
		{
			name:        "host and port",
			addr:        "0.0.0.0:8383",
			wantPort:    8383,
			wantEnabled: true,
		},
		{
			name:        "disabled with zero",
			addr:        "0",
			wantPort:    0,
			wantEnabled: false,
		},
		{
			name:        "empty resolves to controller-runtime default",
			addr:        "",
			wantPort:    defaultBindPort(t),
			wantEnabled: true,
		},
		{
			name:        "missing port falls back to default metricsPort",
			addr:        "127.0.0.1:",
			wantPort:    metricsPort,
			wantEnabled: true,
		},
		{
			name:    "no colon is invalid",
			addr:    "8383",
			wantErr: true,
		},
		{
			name:    "non-numeric port is invalid",
			addr:    ":http",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, enabled, err := metricsPortFromAddr(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("metricsPortFromAddr(%q) = (%d, %t, nil); want error", tt.addr, port, enabled)
				}
				return
			}
			if err != nil {
				t.Fatalf("metricsPortFromAddr(%q) unexpected error: %v", tt.addr, err)
			}
			if port != tt.wantPort {
				t.Errorf("metricsPortFromAddr(%q) port = %d; want %d", tt.addr, port, tt.wantPort)
			}
			if enabled != tt.wantEnabled {
				t.Errorf("metricsPortFromAddr(%q) enabled = %t; want %t", tt.addr, enabled, tt.wantEnabled)
			}
		})
	}
}

// defaultBindPort derives the expected port from controller-runtime's
// DefaultBindAddress so the test stays correct if that default changes.
func defaultBindPort(t *testing.T) int32 {
	t.Helper()
	port, enabled, err := metricsPortFromAddr(metricsserver.DefaultBindAddress)
	if err != nil {
		t.Fatalf("could not derive default bind port from %q: %v", metricsserver.DefaultBindAddress, err)
	}
	if !enabled {
		t.Fatalf("default bind address %q unexpectedly disabled", metricsserver.DefaultBindAddress)
	}
	return port
}
