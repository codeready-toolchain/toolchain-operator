package v1alpha1

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallConfigSpec defines the desired state of InstallConfig
// +k8s:openapi-gen=true
type InstallConfigSpec struct {
	// The configuration required for che operator
	CheOperatorSpec CheOperator `json:"cheOperatorSpec"`
}

type CheOperator struct {
	// The namespace where you want to run che operator
	Namespace string `json:"namespace"`
}

// InstallConfigStatus defines the observed state of InstallConfig
// +k8s:openapi-gen=true
type InstallConfigStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Conditions is an array of current InstallConfig conditions
	// Supported condition types:
	// CreatedCheSubscription, FailedToCreateCheSubscription
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []toolchainv1alpha1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallConfig is the Schema for the installconfigs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type InstallConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallConfigSpec   `json:"spec,omitempty"`
	Status InstallConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallConfigList contains a list of InstallConfig
type InstallConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstallConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstallConfig{}, &InstallConfigList{})
}
