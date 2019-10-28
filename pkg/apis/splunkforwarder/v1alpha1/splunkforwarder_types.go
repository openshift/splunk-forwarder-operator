package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SplunkForwarderSpec defines the desired state of SplunkForwarder
// +k8s:openapi-gen=true
type SplunkForwarderSpec struct {
	SplunkLicenseAccepted bool                    `json:"splunkLicenseAccepted,omitempty"`
	Image                 string                  `json:"image"`
	ImageVersion          string                  `json:"imageTag"`
	SplunkInputs          []SplunkForwarderInputs `json:"splunkInputs"`
}

// SplunkForwarderStatus defines the observed state of SplunkForwarder
// +k8s:openapi-gen=true
type SplunkForwarderStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SplunkForwarder is the Schema for the splunkforwarders API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SplunkForwarder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SplunkForwarderSpec   `json:"spec,omitempty"`
	Status SplunkForwarderStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SplunkForwarderList contains a list of SplunkForwarder
type SplunkForwarderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SplunkForwarder `json:"items"`
}

// SplunkForwarderInputs ia the struct that defines all the splunk inputs
type SplunkForwarderInputs struct {
	Path       string `json:"path"`
	Index      string `json:"index,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
	WhiteList  string `json:"whiteList,omitempty"`
	BlackList  string `json:"blackList,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SplunkForwarder{}, &SplunkForwarderList{})
}
