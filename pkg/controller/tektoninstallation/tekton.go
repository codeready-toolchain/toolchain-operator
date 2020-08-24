package tektoninstallation

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InstallationName the name of the TektonInstallation resource (cluster-scoped)
	InstallationName = "toolchain-tekton-installation"
	// SubscriptionNamespace the namespace of the TekTon Subscription resource
	SubscriptionNamespace = "openshift-operators"
	// SubscriptionName the name for of TekTon Subscription resource
	SubscriptionName = "openshift-pipelines-operator-rh"
	// StartingCSV keeps the CSV version the installation should start with
	StartingCSV = "openshift-pipelines-operator.v1.0.1"
	// TektonConfigName the name of the TektonConfig resource
	TektonConfigName = "cluster"
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
			Channel:                "ocp-4.4",
			Package:                SubscriptionName,
			StartingCSV:            StartingCSV,
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
}

// InstallationSucceeded returns a status condition for the case where the Tekton installation succeeded
func InstallationSucceeded() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   v1alpha1.TektonReady,
		Status: corev1.ConditionTrue,
		Reason: v1alpha1.InstalledReason,
	}
}

// Installing returns a status condition for the case where the Tekton is installing
func Installing(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    v1alpha1.TektonReady,
		Status:  corev1.ConditionFalse,
		Reason:  v1alpha1.InstallingReason,
		Message: message,
	}
}

// InstallationFailed returns a status condition for the case where the Tekton installation failed
func InstallationFailed(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    v1alpha1.TektonReady,
		Status:  corev1.ConditionFalse,
		Reason:  v1alpha1.FailedToInstallReason,
		Message: message,
	}
}

// Unknown returns a status condition for the case where the Tekton installation status is unknown
func Unknown() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   v1alpha1.TektonReady,
		Status: corev1.ConditionFalse,
		Reason: v1alpha1.UnknownReason,
	}
}
