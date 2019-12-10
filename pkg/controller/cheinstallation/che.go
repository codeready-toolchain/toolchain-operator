package cheinstallation

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"

	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InstallationName the name of the CheInstallation resource (cluster-scoped)
	InstallationName = "toolchain-che-installation"
	// Namespace the namespace in which the OLM OperatorGroup and Subscription resources will be created
	Namespace = "toolchain-che"
	// OperatorGroupName the name of the OLM OperatorGroup for Che
	OperatorGroupName = InstallationName
	// SubscriptionName the name of the OLM subscription for Che
	SubscriptionName = "codeready-workspaces"
)

// NewInstallation returns a new CheInstallation resource
func NewInstallation() *v1alpha1.CheInstallation {
	return &v1alpha1.CheInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name: InstallationName, // che installation resource is cluster-scoped, so no namespace is defined
		},
		Spec: v1alpha1.CheInstallationSpec{
			CheOperatorSpec: v1alpha1.CheOperator{
				Namespace: Namespace, // the namespace in which the che operatorgroup and subscription resources will be created
			},
		},
	}
}

// NewSubscription for CodeReady Workspaces operator
func NewSubscription(ns string) *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubscriptionName,
			Namespace: ns,
			Labels:    toolchain.Labels(),
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel:                "latest",
			Package:                "codeready-workspaces",
			StartingCSV:            "crwoperator.v2.0.0",
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
}

// NewNamespace return a new namespace with the toolchain labels
func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: toolchain.Labels(),
		},
	}
}

// NewOperatorGroup returns a new OLM Operator Group for the given ns, with the toolchain labels
func NewOperatorGroup(ns string) *olmv1.OperatorGroup {
	return &olmv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      OperatorGroupName,
			Labels:    toolchain.Labels(),
		},
		Spec: olmv1.OperatorGroupSpec{
			TargetNamespaces: []string{ns},
		},
	}
}

// SubscriptionCreated returns a status condition for the case where the Che installation succeeded
func SubscriptionCreated() toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionCreated(v1alpha1.CheReady, v1alpha1.InstalledReason)
}

// SubscriptionFailed returns a status condition for the case where the Che installation failed
func SubscriptionFailed(message string) toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionFailed(v1alpha1.CheReady, v1alpha1.FailedToInstallReason, message)
}
