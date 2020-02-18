package assert

import (
	"context"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"

	opsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CheInstallationAssertion struct {
	cheInstallation *v1alpha1.CheInstallation
	client          client.Client
	namespacedName  types.NamespacedName
	t               *testing.T
}

func (a *CheInstallationAssertion) loadCheInstallationAssertion() error {
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

// HasFinalizer verifies that the Che installation has the expected finalizer
func (a *CheInstallationAssertion) HasFinalizer(finalizer string) *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)
	assert.Contains(a.t, a.cheInstallation.ObjectMeta.GetFinalizers(), finalizer)
	return a
}

// HasNoFinalizer verifies that the Che installation has the no finalizer
func (a *CheInstallationAssertion) HasNoFinalizer() *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)
	assert.Empty(a.t, a.cheInstallation.ObjectMeta.GetFinalizers())
	return a
}

// HasOwnerRef verifies that the Che installation has the expected ownerReference
func (a *CheInstallationAssertion) HasOwnerRef(sub *opsv1alpha1.Subscription) *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)

	references := a.cheInstallation.ObjectMeta.GetOwnerReferences()
	assertThatContainsOwnerReference(a.t, a.client, references, sub)
	return a
}

// HasNoOwnerRef verifies that the Che installation has no ownerReference
func (a *CheInstallationAssertion) HasNoOwnerRef() *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)

	references := a.cheInstallation.ObjectMeta.GetOwnerReferences()
	assert.Empty(a.t, references)
	return a
}

func (a *CheInstallationAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)
	AssertConditionsMatch(a.t, a.cheInstallation.Status.Conditions, expected...)
	return a
}

func (a *CheInstallationAssertion) HasNoCondition() *CheInstallationAssertion {
	err := a.loadCheInstallationAssertion()
	require.NoError(a.t, err)
	AssertConditionsMatch(a.t, a.cheInstallation.Status.Conditions)
	return a
}
