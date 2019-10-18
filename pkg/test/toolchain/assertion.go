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

type InstallConfigAssertion struct {
	installConfig  *v1alpha1.InstallConfig
	client         client.Client
	namespacedName types.NamespacedName
	t              *testing.T
}

func (a *InstallConfigAssertion) loadInstallConfigAssertion() error {
	if a.installConfig != nil {
		return nil
	}
	ic := &v1alpha1.InstallConfig{}
	err := a.client.Get(context.TODO(), a.namespacedName, ic)
	a.installConfig = ic
	return err
}

func AssertThatInstallConfig(t *testing.T, ns, name string, client client.Client) *InstallConfigAssertion {
	return &InstallConfigAssertion{
		client:         client,
		namespacedName: types.NamespacedName{ns, name},
		t:              t,
	}
}

func (a *InstallConfigAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *InstallConfigAssertion {
	err := a.loadInstallConfigAssertion()
	require.NoError(a.t, err)
	AssertConditionsMatch(a.t, a.installConfig.Status.Conditions, expected...)
	return a
}
