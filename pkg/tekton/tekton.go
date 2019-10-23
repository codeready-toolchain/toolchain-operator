package tekton

import (
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SubscriptionNamespace = "openshift-operators"
	SubscriptionName      = "openshift-pipelines-operator"
	SubscriptionSuccess   = "tekton operator subscription created"
)

//NewSubscription for openshift-pipeline operator
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
			StartingCSV:            "openshift-pipelines-operator.v0.5.2",
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
}

func SubscriptionFailed(message string) toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionFailed(v1alpha1.TektonReady, v1alpha1.FailedToInstallReason, message)
}

func SubscriptionCreated(message string) toolchainv1alpha1.Condition {
	return v1alpha1.SubscriptionCreated(v1alpha1.TektonReady, v1alpha1.InstalledReason, message)
}
