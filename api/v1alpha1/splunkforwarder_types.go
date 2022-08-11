/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SplunkForwarderSpec defines the desired state of SplunkForwarder
// +k8s:openapi-gen=true
type SplunkForwarderSpec struct {
	// Adds an --accept-license flag to automatically accept the Splunk License Agreement.
	// Must be true for the Red Hat provided Splunk Forwarder image.
	// Optional: Defaults to false.
	SplunkLicenseAccepted bool `json:"splunkLicenseAccepted,omitempty"`
	// Container image path to the Splunk Forwarder
	Image string `json:"image"`
	// The container image tag of the Splunk Forwarder image.
	// Is not used if ImageDigest is supplied.
	// Optional: Defaults to latest
	ImageTag string `json:"imageTag,omitempty"`
	// Container image digest of the Splunk Forwarder image.
	// Has precedence and is recommended over ImageTag.
	// Optional: Defaults to latest
	ImageDigest string `json:"imageDigest,omitempty"`
	// Unique cluster name.
	// Optional: Looked up on the cluster if not provided, default to openshift
	ClusterID string `json:"clusterID,omitempty"`
	// +listType=atomic
	SplunkInputs []SplunkForwarderInputs `json:"splunkInputs"`
	// Whether an additional Splunk Heavy Forwarder should be deployed.
	// Optional: Defaults to false.
	UseHeavyForwarder bool `json:"useHeavyForwarder,omitempty"`
	// Container image path to the Splunk Heavy Forwarder image. Required when
	// UseHeavyForwarder is true.
	HeavyForwarderImage string `json:"heavyForwarderImage,omitempty"`
	// Container image digest of the container image defined in HeavyForwarderImage.
	// Optional: Defaults to latest
	HeavyForwarderDigest string `json:"heavyForwarderDigest,omitempty"`
	// Number of desired Splunk Heavy Forwarder pods.
	// Optional: Defaults to 2
	HeavyForwarderReplicas int32 `json:"heavyForwarderReplicas,omitempty"`
	// Specifies the value of the NodeSelector for the Splunk Heavy Forwarder pods
	// with key: "node-role.kubernetes.io"
	// Optional: Defaults to an empty value.
	HeavyForwarderSelector string `json:"heavyForwarderSelector,omitempty"`
	// List of additional filters supplied to configure the Splunk Heavy Forwarder
	// Optional: Defaults to no additional filters (no transforms.conf).
	// +listType=map
	// +listMapKey=name
	Filters []SplunkFilter `json:"filters,omitempty"`
}

// SplunkForwarderStatus defines the observed state of SplunkForwarder
// +k8s:openapi-gen=true
type SplunkForwarderStatus struct {
}

// +kubebuilder:object:root=true

// SplunkForwarder is the Schema for the splunkforwarders API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SplunkForwarder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SplunkForwarderSpec   `json:"spec,omitempty"`
	Status SplunkForwarderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SplunkForwarderList contains a list of SplunkForwarder
type SplunkForwarderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SplunkForwarder `json:"items"`
}

// SplunkFilter is the struct that configures Splunk Heavy Forwarder filters.
type SplunkFilter struct {
	// Name of the filter, will be prepended with "filter_".
	Name string `json:"name"`
	// Routing criteria regex for the filter to match on.
	Filter string `json:"filter"`
}

// SplunkForwarderInputs is the struct that defines all the splunk inputs
type SplunkForwarderInputs struct {
	// Required: Filepath for Splunk to monitor.
	Path string `json:"path"`
	// Repository for data. More info: https://docs.splunk.com/Splexicon:Index
	// Optional: Defaults to "main"
	Index string `json:"index,omitempty"`
	// Data structure of the event. More info: https://docs.splunk.com/Splexicon:Sourcetype
	// Optional: Defaults to "_json"
	SourceType string `json:"sourceType,omitempty"`
	// Regex to monitor certain files. Multiple regex rules may be specified separated by "|" (OR)
	// Optional: Defaults to monitoring all files in the specified Path
	WhiteList string `json:"whiteList,omitempty"`
	// Regex to exclude certain files from monitoring. Multiple regex rules may be specified separated by "|" (OR)
	// Optional: Defaults to monitoring all files in the specified Path
	BlackList string `json:"blackList,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SplunkForwarder{}, &SplunkForwarderList{})
}
