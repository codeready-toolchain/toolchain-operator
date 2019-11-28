package che

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
	SubscriptionName = "codeready-workspaces"
	CheClusterName   = "codeready-workspaces"
	CheFlavorName    = "codeready"
)

//NewSubscription for eclipse Che operator
func NewSubscription(ns string) *olmv1alpha1.Subscription {
	/* 	Default Subscription yaml: oc get sub codeready-workspaces -o yaml
	apiVersion: operators.coreos.com/v1alpha1
	kind: Subscription
	metadata:
	  creationTimestamp: "2019-11-28T04:47:12Z"
	  generation: 1
	  name: codeready-workspaces
	  namespace: demo-crw
	  resourceVersion: "30249"
	  selfLink: /apis/operators.coreos.com/v1alpha1/namespaces/demo-crw/subscriptions/codeready-workspaces
	  uid: 24d3ecab-119a-11ea-9fce-52fdfc072182
	spec:
	  channel: latest
	  installPlanApproval: Automatic
	  name: codeready-workspaces
	  source: redhat-operators
	  sourceNamespace: openshift-marketplace
	  startingCSV: crwoperator.v2.0.0
	*/
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
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			StartingCSV:            "crwoperator.v2.0.0",
		},
	}
}

func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: toolchain.Labels(),
		},
	}
}

func NewOperatorGroup(ns string) *olmv1.OperatorGroup {
	return &olmv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: ns,
			Labels:    toolchain.Labels(),
		},
		Spec: olmv1.OperatorGroupSpec{

			TargetNamespaces: []string{ns},
		},
	}
}

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
				TlsSupport:     false, // TODO VN: change tls_support to true
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

func SubscriptionFailed(message string) toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionFailed(v1alpha1.CheReady, v1alpha1.FailedToInstallReason, message)
}

func SubscriptionCreated() toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionCreated(v1alpha1.CheReady, v1alpha1.InstalledReason)
}
