package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	configv1 "github.com/openshift/api/config/v1"
	opmetrics "github.com/openshift/operator-custom-metrics/pkg/metrics"
	splunkforwarderv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	"github.com/openshift/splunk-forwarder-operator/controllers/secret"
	"github.com/openshift/splunk-forwarder-operator/controllers/splunkforwarder"
	"github.com/openshift/splunk-forwarder-operator/version"
	"github.com/operator-framework/operator-lib/leader"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monclientv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	//+kubebuilder:scaffold:imports
)

const (
	// Environment variable to determine operator run mode
	ForceRunModeEnv = "OSDK_FORCE_RUN_MODE"
	// Flags that the operator is running locally
	LocalRunMode = "local"
)

var (
	metricsPort int32 = 8383
	scheme            = apiruntime.NewScheme()
	setupLog          = ctrl.Log.WithName("setup")
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", version.SDKVersion))
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(splunkforwarderv1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1.Install(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":"+fmt.Sprintf("%d", metricsPort), "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	printVersion()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f48c54da.managed.openshift.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	// This doesn't work locally, so only perform it when running on-cluster
	if strings.ToLower(os.Getenv(ForceRunModeEnv)) != LocalRunMode {
		err = leader.Become(ctx, "splunk-forwarder-operator-lock")
		if err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	} else {
		setupLog.Info("bypassing leader election due to local execution")
	}

	// Derive the metrics Service port from the configured bind address so the
	// Service always targets the port the metrics server actually listens on.
	servicePort, metricsEnabled, err := metricsPortFromAddr(metricsAddr)
	if err != nil {
		setupLog.Error(err, "invalid metrics-bind-address", "address", metricsAddr)
		os.Exit(1)
	}

	// Add the Metrics Service, unless metrics are disabled (BindAddress "0").
	if metricsEnabled {
		if err := addMetrics(ctx, mgr.GetClient(), mgr.GetConfig(), servicePort, false); err != nil {
			log.Error(err, "Metrics service is not added.")
			os.Exit(1)
		}
	} else {
		setupLog.Info("metrics server disabled, skipping metrics Service creation")
	}

	// Add SplunkForwarder controller to manager
	if err = (&splunkforwarder.SplunkForwarderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SplunkForwarder")
		os.Exit(1)
	}

	// Add Secret controller to manager
	if err = (&secret.SecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// metricsPortFromAddr extracts the TCP port from a metrics bind address such as
// ":8383" or "0.0.0.0:8383" so the metrics Service can target the same port the
// metrics server listens on. It mirrors controller-runtime's metrics server
// address semantics: a bind address of "0" disables the metrics server
// (enabled=false, and no Service should be created), and an empty address is
// resolved to metricsserver.DefaultBindAddress before the port is extracted.
func metricsPortFromAddr(addr string) (port int32, enabled bool, err error) {
	// "0" disables the metrics server in controller-runtime; no Service needed.
	if addr == "0" {
		return 0, false, nil
	}
	// An empty address resolves to the controller-runtime default.
	if addr == "" {
		addr = metricsserver.DefaultBindAddress
	}
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, false, fmt.Errorf("could not parse metrics bind address %q: %w", addr, err)
	}
	if portStr == "" {
		return metricsPort, true, nil
	}
	parsed, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return 0, false, fmt.Errorf("invalid metrics port in bind address %q: %w", addr, err)
	}
	return int32(parsed), true, nil
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cl client.Client, cfg *rest.Config, port int32, withServiceMonitor bool) error {
	service, err := opmetrics.GenerateService(port, "http-metrics", config.OperatorName+"-metrics", config.OperatorNamespace, map[string]string{"name": config.OperatorName})
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
		return err
	}

	log.Info(fmt.Sprintf("Attempting to create service %s", service.Name))
	err = cl.Create(ctx, service)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Error(err, "Could not create metrics service")
			return err
		} else {
			log.Info("Metrics service already exists, will not create")
		}
	}

	// If there's no need to create a ServiceMonitor, just return
	if !withServiceMonitor {
		return nil
	}

	sm := opmetrics.GenerateServiceMonitor(service)
	// ErrSMMetricsExists is used to detect if the -metrics ServiceMonitor already exists
	var ErrSMMetricsExists = fmt.Sprintf("servicemonitors.monitoring.coreos.com \"%s-metrics\" already exists", config.OperatorName)
	log.Info(fmt.Sprintf("Attempting to create service monitor %s", sm.Name))
	mclient := monclientv1.NewForConfigOrDie(cfg)
	_, err = mclient.ServiceMonitors(config.OperatorNamespace).Create(ctx, sm, metav1.CreateOptions{})
	if err != nil {
		if err.Error() != ErrSMMetricsExists {
			return err
		}
		log.Info("ServiceMonitor already exists")
	}
	log.Info(fmt.Sprintf("Successfully configured service monitor %s", sm.Name))

	return nil
}
