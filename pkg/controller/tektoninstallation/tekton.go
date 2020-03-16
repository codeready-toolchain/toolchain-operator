package tektoninstallation

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	config "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"

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
	SubscriptionName = "openshift-pipelines-operator"
	// StartingCSV keeps the CSV version the installation should start with
	StartingCSV = "openshift-pipelines-operator.v0.8.2"
	// TektonClusterName the name of the TektonCluster resource
	TektonClusterName = "cluster"
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

// InstallationSucceeded returns a status condition for the case where the Tekton installation succeeded
func InstallationSucceeded() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   v1alpha1.TektonReady,
		Status: corev1.ConditionTrue,
		Reason: v1alpha1.InstalledReason,
	}
}

// InstallationInstalling returns a status condition for the case where the Tekton is installing
func InstallationInstalling() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   v1alpha1.TektonReady,
		Status: corev1.ConditionFalse,
		Reason: v1alpha1.InstallingReason,
	}
}

func InstallationSubscriptionCreated() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    v1alpha1.TektonReady,
		Status:  corev1.ConditionFalse,
		Reason:  v1alpha1.InstallingReason,
		Message: "Subscription created",
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

// InstallationUnknown returns a status condition for the case where the Tekton installation status is unknown
func InstallationUnknown() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   v1alpha1.TektonReady,
		Status: corev1.ConditionFalse,
		Reason: v1alpha1.UnknownReason,
	}
}

// NewTektonCluster returns a new TektonCluster with the given conditions
func NewTektonCluster(conditions ...config.ConfigCondition) *config.Config {
	return &config.Config{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:   TektonClusterName,
			Labels: toolchain.Labels(),
		},
		Spec: config.ConfigSpec{
			TargetNamespace: "",
		},
		Status: config.ConfigStatus{
			Conditions: conditions,
		},
	}
}
