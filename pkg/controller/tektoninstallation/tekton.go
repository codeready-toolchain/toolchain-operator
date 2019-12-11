package tektoninstallation

import (
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InstallationName the name of the TektonInstallation resource (cluster-scoped)
	InstallationName = "toolchain-tekton-installation"
	// SubscriptionNamespace the namespace of the TekTon Subscription resource
	SubscriptionNamespace = "openshift-operators"
	// SubscriptionName the name for of TekTon Subscription resource
	SubscriptionName = "openshift-pipelines-operator"
	// StartingCSV keeps the CSV version the installation should start with
	StartingCSV = "openshift-pipelines-operator.v0.8.1"
)

// NewInstallation returns a new TektonInstallation resource
func NewInstallation() *v1alpha1.TektonInstallation {
	return &v1alpha1.TektonInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name: InstallationName, // Tekton installation resource is cluster-scoped, so no namespace is defined
		},
	}
}

// NewSubscription for openshift-pipeline operator
func NewSubscription(ns string) *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubscriptionName,
			Namespace: ns,
			Labels:    toolchain.Labels(),
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel:                "dev-preview",
			Package:                "openshift-pipelines-operator",
			StartingCSV:            StartingCSV,
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
}

// SubscriptionCreated returns a status condition for the case where the Tekton installation succeeded
func SubscriptionCreated() toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionCreated(v1alpha1.TektonReady, v1alpha1.InstalledReason)
}

// SubscriptionFailed returns a status condition for the case where the Tekton installation failed
func SubscriptionFailed(message string) toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionFailed(v1alpha1.TektonReady, v1alpha1.FailedToInstallReason, message)
}
