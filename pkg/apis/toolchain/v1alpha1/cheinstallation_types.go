package v1alpha1

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheInstallationSpec defines the desired state of CheInstallation
// +k8s:openapi-gen=true
type CheInstallationSpec struct {
	// The configuration required for Che operator
	CheOperatorSpec CheOperator `json:"cheOperatorSpec"`
}

type CheOperator struct {
	// The namespace where the CodeReady Workspaces operator will be installed
	Namespace string `json:"namespace"`
}

// CheInstallationStatus defines the observed state of CheInstallation
// +k8s:openapi-gen=true
type CheInstallationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// +optional

	// CheServerURL the URL of the Che Server, once the installation completed
	// +optional
	CheServerURL string `json:"CheServerURL,omitempty"`

	// Conditions is an array of current CheInstallation conditions
	// Supported condition types:
	// CheReady
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType
	Conditions []toolchainv1alpha1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CheInstallation is the Schema for the cheinstallations API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cheinstallations,scope=Cluster
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"CheReady\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"CheReady\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"CheReady\")].message",priority=1
type CheInstallation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CheInstallationSpec   `json:"spec,omitempty"`
	Status CheInstallationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CheInstallationList contains a list of CheInstallation
type CheInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CheInstallation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CheInstallation{}, &CheInstallationList{})
}
