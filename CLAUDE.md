# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes operator that manages Splunk Universal Forwarder deployments. It creates DaemonSets that deploy Splunk forwarder pods on each node in the cluster to collect and forward logs to Splunk instances.

## Key Architecture Components

- **API Types**: `api/v1alpha1/splunkforwarder_types.go` - Defines the SplunkForwarder CRD with fields for image configuration, cluster ID, and input specifications
- **Controllers**:
  - `controllers/splunkforwarder/splunkforwarder_controller.go` - Main controller reconciling SplunkForwarder resources
  - `controllers/secret/secret_controller.go` - Manages Splunk authentication secrets
- **Kubernetes Resources**: `pkg/kube/` - Utilities for creating DaemonSets, ConfigMaps, Services, Volumes, and VolumeMounts
- **Configuration**: Uses a "splunk-auth" secret containing `cacert.pem`, `server.pem`, `outputs.conf`, and `limits.conf`

## Development Commands

**Building and Testing:**
```bash
# Build the operator binary
make go-build

# Run unit tests
make go-test

# Run linting and validation
make lint
make validate

# Build container image
make docker-build

# Build and push images (for releases)
make build-push
```

**Local Development:**
```bash
# Required environment variables for local runs
export OPERATOR_NAMESPACE=openshift-splunk-forwarder-operator
export WATCH_NAMESPACE=""
export OSDK_FORCE_RUN_MODE="local"
```

**Image Updates:**
```bash
# Update to latest forwarder image from upstream
make image-update

# Update to specific version/commit
make SFI_UPDATE=<commit/branch> image-update

# Update OLM templates with new image digests
make image-digests
```

**Container Testing:**
```bash
# Run tests in boilerplate container
make container-test

# Run linting in container
make container-lint
```

## Project Structure

- Built with Operator SDK v1.21.0+ and Go 1.23+
- Uses OpenShift boilerplate for standardized build/test/deploy workflows
- Includes E2E tests in `test/e2e/`
- FIPS-compliant builds supported via `FIPS_ENABLED=true`
- Supports both Universal Forwarder and Heavy Forwarder deployments