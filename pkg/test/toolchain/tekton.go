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

type TektonInstallationAssertion struct {
	tektonInstallation *v1alpha1.TektonInstallation
	client             client.Client
	namespacedName     types.NamespacedName
	t                  *testing.T
}

func (a *TektonInstallationAssertion) loadTektonInstallationAssertion() error {
	if a.tektonInstallation != nil {
		return nil
	}
	ti := &v1alpha1.TektonInstallation{}
	err := a.client.Get(context.TODO(), a.namespacedName, ti)
	a.tektonInstallation = ti
	return err
}

func AssertThatTektonInstallation(t *testing.T, ns, name string, client client.Client) *TektonInstallationAssertion {
	return &TektonInstallationAssertion{
		client:         client,
		namespacedName: types.NamespacedName{ns, name},
		t:              t,
	}
}

func (a *TektonInstallationAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *TektonInstallationAssertion {
	err := a.loadTektonInstallationAssertion()
	require.NoError(a.t, err)
	AssertConditionsMatch(a.t, a.tektonInstallation.Status.Conditions, expected...)
	return a
}
