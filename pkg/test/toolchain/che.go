package toolchain

import (
	"context"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type CheInstallationAssertion struct {
	cheInstallation *v1alpha1.CheInstallation
	client          client.Client
	namespacedName  types.NamespacedName
	t               *testing.T
}

func (a *CheInstallationAssertion) loadCheInstallationAssertion() error {
	if a.cheInstallation != nil {
		return nil
	}
	ic := &v1alpha1.CheInstallation{}
	err := a.client.Get(context.TODO(), a.namespacedName, ic)
	a.cheInstallation = ic
	return err
}

func AssertThatCheInstallation(t *testing.T, ns, name string, client client.Client) *CheInstallationAssertion {
	return &CheInstallationAssertion{
		client:         client,
		namespacedName: types.NamespacedName{ns, name},
		t:              t,
	}
}

func (a *CheInstallationAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)
	AssertConditionsMatch(a.t, a.cheInstallation.Status.Conditions, expected...)
	return a
}
