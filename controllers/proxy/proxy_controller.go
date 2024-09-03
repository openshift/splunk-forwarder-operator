package proxy

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
)

var log = logf.Log.WithName("controller_proxy")

type ProxyReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Config *corev1.ConfigMap
}

func (r *ProxyReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
    
	proxy := &configv1.Proxy{}
	if err := r.Client.Get(ctx, config.ProxyName, proxy); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	proxyConfig := &corev1.ConfigMap{}

	// Fetch the proxy config
	proxyConfig.Name = config.ProxyConfigMapName
	proxyConfig.Data = make(map[string]string)

    // splunkforwarder only supports http proxies, so we need to check for https
	if proxy.Status.HTTPProxy != "" && strings.HasPrefix(proxy.Status.HTTPProxy, "http://") {
		proxyConfig.Data["HTTP_PROXY"] = proxy.Status.HTTPProxy
	}

    // splunkforwarder only supports http proxies, so we need to check for https
	if proxy.Status.HTTPSProxy != "" && strings.HasPrefix(proxy.Status.HTTPSProxy, "http://") {
		proxyConfig.Data["HTTPS_PROXY"] = proxy.Status.HTTPSProxy
	}

	if len(proxyConfig.Data) > 0 && proxy.Status.NoProxy != "" {
		proxyConfig.Data["NO_PROXY"] = proxy.Status.NoProxy
	}

	// Fetch the trusted CA bundle
	trustedCa := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, config.TrustedCABundleName, trustedCa); err != nil {
        return reconcile.Result{Requeue: true}, err
	}

	forwarders := &v1alpha1.SplunkForwarderList{}
	if err := r.Client.List(ctx, forwarders); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	var lastErr error

	for _, forwarder := range forwarders.Items {
		ca := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: config.TrustedCABundleName.Name, Namespace: forwarder.Namespace}}
		ca.Data = trustedCa.Data

		proxyConfig.Namespace = forwarder.Namespace

		if err := r.Client.Update(ctx, ca); err != nil {
			if err := r.Client.Create(ctx, ca); err != nil {
				lastErr = err
			}
		}
		log.Info("Updated trusted CA bundle", "namespace", forwarder.Namespace)
		if err := r.Client.Update(ctx, proxyConfig); err != nil {
			if err := r.Client.Create(ctx, proxyConfig); err != nil {
				lastErr = err
			}
		}
		log.Info("Updated proxy config", "namespace", forwarder.Namespace)
	}
    

    if err := lastErr; err != nil {
        log.Error(err, "reconcile failed")
        return reconcile.Result{Requeue: true}, err

    }

	return reconcile.Result{}, lastErr

}


// SetupWithManager sets up the controller with the Manager.
func (r *ProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).Named("cluster-proxy").For(&configv1.Proxy{}).Complete(r)
}
