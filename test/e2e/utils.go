//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/osde2e-common/pkg/clients/openshift"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
)

// Create test splunkforwarder CR definition
func makeMinimalSplunkforwarder(name string, namespace string) sfv1alpha1.SplunkForwarder {
	return sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			SplunkLicenseAccepted: true,
			UseHeavyForwarder:     false,
			SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
				{
					Path: "",
				},
			},
		},
	}
}

// makeSplunkforwarderWithIndex creates a test SplunkForwarder CR with custom index settings
func makeSplunkforwarderWithIndex(name, namespace, path, index, sourcetype string) sfv1alpha1.SplunkForwarder {
	return sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			SplunkLicenseAccepted: true,
			Image:                 "quay.io/redhat-services-prod/openshift/splunk-forwarder-images",
			ImageTag:              "latest",
			SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
				{
					Path:       path,
					Index:      index,
					SourceType: sourcetype,
				},
			},
		},
	}
}

// createTestSecrets creates splunk-auth and splunk-hec-token secrets for testing
func createTestSecrets(ctx context.Context, k8s *openshift.Client, namespace string) error {
	// Create splunk-auth secret (mTLS mode)
	authSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkAuthSecretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"cacert.pem":   generateSelfSignedCACert(),
			"server.pem":   generateSelfSignedServerCert(),
			"outputs.conf": generateMTLSOutputsConf(),
			"limits.conf":  generateLimitsConf(),
		},
	}

	if err := k8s.Create(ctx, authSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// Create splunk-hec-token secret (HEC mode)
	hecSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkHECTokenSecretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"outputs.conf": generateHECOutputsConf(),
		},
	}

	if err := k8s.Create(ctx, hecSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createTestSecrets creates splunk-auth and splunk-hec-token secrets for testing
func restoreSplunkAuthSecret(ctx context.Context, k8s *openshift.Client, namespace string) error {
	// Create splunk-auth secret (mTLS mode)
	authSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkAuthSecretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"cacert.pem":   generateSelfSignedCACert(),
			"server.pem":   generateSelfSignedServerCert(),
			"outputs.conf": generateMTLSOutputsConf(),
			"limits.conf":  generateLimitsConf(),
		},
	}

	if err := k8s.Create(ctx, authSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// cleanupTestSecrets removes test secrets
func cleanupTestSecrets(ctx context.Context, k8s *openshift.Client, namespace string) {
	// Delete splunk-auth secret, ignore not found errors
	authSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkAuthSecretName,
			Namespace: namespace,
		},
	}
	k8s.Delete(ctx, authSecret)

	// Delete splunk-hec-token secret, ignore not found errors
	hecSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkHECTokenSecretName,
			Namespace: namespace,
		},
	}
	k8s.Delete(ctx, hecSecret)
}

// generateSelfSignedCACert dynamically generates a PEM-encoded self-signed CA certificate
// using Go's crypto packages for realistic testing
func generateSelfSignedCACert() string {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("failed to generate CA private key: %v", err))
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Splunk E2E Test CA"},
			Country:      []string{"US"},
			CommonName:   "Splunk Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(30 * 24 * time.Hour), // Valid for 1 month
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(fmt.Sprintf("failed to create CA certificate: %v", err))
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return string(certPEM)
}

// generateSelfSignedServerCert dynamically generates a PEM-encoded server certificate
// with both certificate and private key for Splunk mTLS authentication
func generateSelfSignedServerCert() string {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("failed to generate server private key: %v", err))
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Splunk E2E Test Server"},
			Country:      []string{"US"},
			CommonName:   "splunk-forwarder-test.example.com",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(30 * 24 * time.Hour), // Valid for 1 month
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:    []string{"splunk-forwarder-test.example.com", "localhost"},
	}

	// Create self-signed certificate (in production, this would be signed by CA)
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(fmt.Sprintf("failed to create server certificate: %v", err))
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Return combined certificate and private key
	return string(certPEM) + string(privateKeyPEM)
}

// generateMTLSOutputsConf returns outputs.conf content for mTLS mode
func generateMTLSOutputsConf() string {
	return `[tcpout]
defaultGroup = default-autolb-group

[tcpout:default-autolb-group]
server = splunk-forwarder-test.example.com:9997
clientCert = $SPLUNK_HOME/etc/apps/splunkauth/default/server.pem
sslCertPath = $SPLUNK_HOME/etc/apps/splunkauth/default/server.pem
sslRootCAPath = $SPLUNK_HOME/etc/apps/splunkauth/default/cacert.pem
sslPassword = test
sslVerifyServerCert = false`
}

// generateHECOutputsConf returns outputs.conf content for HEC mode
// with a realistic HEC token format (UUID) for testing
func generateHECOutputsConf() string {
	return `[http]
httpEventCollectorToken = 12345678-1234-5678-1234-567812345678
uri = https://splunk-hec.example.com:8088/services/collector/event
disabled = 0
useACK = true
token = 12345678-1234-5678-1234-567812345678

[tcpout]
defaultGroup = nothing
disabled = true`
}

// generateLimitsConf returns limits.conf content for testing
func generateLimitsConf() string {
	return `[thruput]
maxKBps = 0`
}

// grantSCCAccess adds the default service account from the specified namespace
// to the splunkforwarder SCC to allow test DaemonSets to run with required privileges
func grantSCCAccess(ctx context.Context, k8s *openshift.Client, namespace string) error {
	sccName := "splunkforwarder"
	serviceAccount := fmt.Sprintf("system:serviceaccount:%s:default", namespace)

	// Register the security API scheme
	if err := securityv1.Install(k8s.GetScheme()); err != nil {
		return fmt.Errorf("failed to register security API: %w", err)
	}

	var scc securityv1.SecurityContextConstraints
	err := k8s.Get(ctx, sccName, "", &scc)
	if err != nil {
		return fmt.Errorf("failed to get SCC %s: %w", sccName, err)
	}

	// Check if service account is already in the users list
	for _, user := range scc.Users {
		if user == serviceAccount {
			// Already granted, nothing to do
			return nil
		}
	}

	// Add service account to SCC users
	scc.Users = append(scc.Users, serviceAccount)

	err = k8s.Update(ctx, &scc)
	if err != nil {
		return fmt.Errorf("failed to update SCC %s: %w", sccName, err)
	}

	return nil
}

// revokeSCCAccess removes the default service account from the specified namespace
// from the splunkforwarder SCC
func revokeSCCAccess(ctx context.Context, k8s *openshift.Client, namespace string) error {
	sccName := "splunkforwarder"
	serviceAccount := fmt.Sprintf("system:serviceaccount:%s:default", namespace)

	var scc securityv1.SecurityContextConstraints
	err := k8s.Get(ctx, sccName, "", &scc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// SCC doesn't exist, nothing to revoke
			return nil
		}
		return fmt.Errorf("failed to get SCC %s: %w", sccName, err)
	}

	// Remove service account from SCC users
	updatedUsers := make([]string, 0, len(scc.Users))
	for _, user := range scc.Users {
		if user != serviceAccount {
			updatedUsers = append(updatedUsers, user)
		}
	}

	// Only update if we actually removed something
	if len(updatedUsers) < len(scc.Users) {
		scc.Users = updatedUsers
		err = k8s.Update(ctx, &scc)
		if err != nil {
			return fmt.Errorf("failed to update SCC %s: %w", sccName, err)
		}
	}

	return nil
}
