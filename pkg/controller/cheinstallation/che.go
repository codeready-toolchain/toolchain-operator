package cheinstallation

import (
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"

	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
)

const (
	// InstallationName the name of the CheInstallation resource (cluster-scoped)
	InstallationName = "toolchain-workspaces-installation"
	// Namespace the namespace in which the OLM OperatorGroup and Subscription resources will be created
	Namespace = "toolchain-workspaces"
	// OperatorGroupName the name of the OLM OperatorGroup for Che
	OperatorGroupName = InstallationName
	// SubscriptionName the name of the OLM subscription for Che
	SubscriptionName = "codeready-workspaces"
	// StartingCSV keeps the CSV version the installation should start with
	StartingCSV = "crwoperator.v2.0.0"
	// CheClusterName the name of the CheCluster
	CheClusterName = "codeready-workspaces"
	// CheFlavorName the name of the CheCluster flavor
	CheFlavorName = "codeready"
	// AvailableStatus constant for Available status
	AvailableStatus = "Available"
	// CheClusterCRDName the fully qualified name of the CheCluster CRD
	CheClusterCRDName = "checlusters.org.eclipse.che"
)

// NewInstallation returns a new CheInstallation resource
func NewInstallation() *v1alpha1.CheInstallation {
	return &v1alpha1.CheInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       InstallationName, // Che installation resource is cluster-scoped, so no namespace is defined
			Finalizers: []string{toolchainv1alpha1.FinalizerName},
		},
		Spec: v1alpha1.CheInstallationSpec{
			CheOperatorSpec: v1alpha1.CheOperator{
				Namespace: Namespace, // the namespace in which the Che operatorgroup and subscription resources will be created
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
			InstallPlanApproval:    olmv1alpha1.ApprovalAutomatic,
			Package:                "codeready-workspaces",
			StartingCSV:            StartingCSV,
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

// NewCheCluster returns a nee CheCluster with the given namespace
func NewCheCluster(ns string) *orgv1.CheCluster {
	return &orgv1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CheClusterName,
			Namespace: ns,
			Labels:    toolchain.Labels(),
		},

		Spec: orgv1.CheClusterSpec{
			Server: orgv1.CheClusterSpecServer{
				CheFlavor:      CheFlavorName,
				TlsSupport:     false,
				SelfSignedCert: false,
			},

			Database: orgv1.CheClusterSpecDB{
				ExternalDb: false,
			},

			Auth: orgv1.CheClusterSpecAuth{
				OpenShiftoAuth:           true,
				ExternalIdentityProvider: false,
			},

			Storage: orgv1.CheClusterSpecStorage{
				PvcStrategy:       "per-workspace",
				PvcClaimSize:      "1Gi",
				PreCreateSubPaths: true,
			},
		},
	}
}

// Installing returns the status condition to set when Che is (still) being installed
func Installing(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    v1alpha1.CheReady,
		Status:  v1.ConditionFalse,
		Reason:  v1alpha1.InstallingReason,
		Message: message,
	}
}

// Terminating returns the status condition to set when Che is (still) being uninstalled
func Terminating(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    v1alpha1.CheReady,
		Status:  v1.ConditionFalse,
		Reason:  v1alpha1.TerminatingReason,
		Message: message,
	}
}

// InstallationSucceeded returns a status condition for the case where the Che installation succeeded
func InstallationSucceeded() toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:   v1alpha1.CheReady,
		Status: v1.ConditionTrue,
		Reason: v1alpha1.InstalledReason,
	}
}

// InstallationFailed returns a status condition for the case where the Che installation failed
func InstallationFailed(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    v1alpha1.CheReady,
		Status:  v1.ConditionFalse,
		Reason:  v1alpha1.FailedToInstallReason,
		Message: message,
	}
}
