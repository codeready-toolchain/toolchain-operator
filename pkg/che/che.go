package che

import (
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//NewSubscription for eclipse Che operator
func NewSubscription(ns string) *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eclipse-che",
			Namespace: ns,
			Labels:    Labels(),
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel:                "stable",
			Package:                "eclipse-che",
			StartingCSV:            "eclipse-che.v7.2.0",
			CatalogSource:          "community-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
}

func Labels() map[string]string {
	return map[string]string{"provider": "toolchain-operator"}
}

func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: Labels(),
		},
	}
}

func NewOperatorGroup(ns string) *olmv1.OperatorGroup {
	return &olmv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ns,
			GenerateName: ns,
			Labels:       Labels(),
		},
		Spec: olmv1.OperatorGroupSpec{

			TargetNamespaces: []string{ns},
		},
	}
}
