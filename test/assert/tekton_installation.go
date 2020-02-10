package assert

import (
	"context"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"

	opsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TektonInstallationAssertion an assertion on the Tekton installation
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

// AssertThatTektonInstallation return an assertion on the Tekton installation
func AssertThatTektonInstallation(t *testing.T, ns, name string, client client.Client) *TektonInstallationAssertion {
	return &TektonInstallationAssertion{
		client: client,
		namespacedName: types.NamespacedName{
			Namespace: ns,
			Name:      name,
		},
		t: t,
	}
}

// HasOwnerRef verifies that the Tekton installation has the expected ownerReference
func (a *TektonInstallationAssertion) HasOwnerRef(sub *opsv1alpha1.Subscription) *TektonInstallationAssertion {
	err := a.loadTektonInstallationAssertion()
	require.NoError(a.t, err)

	references := a.tektonInstallation.ObjectMeta.GetOwnerReferences()
	assertThatContainsOwnerReference(a.t, a.client, references, sub)
	return a
}

// HasNoOwnerRef verifies that the Tekton installation has no ownerReference
func (a *TektonInstallationAssertion) HasNoOwnerRef() *TektonInstallationAssertion {
	err := a.loadTektonInstallationAssertion()
	require.NoError(a.t, err)

	references := a.tektonInstallation.ObjectMeta.GetOwnerReferences()
	assert.Empty(a.t, references)
	return a
}

// HasConditions verifies that the Tekton installation has the expected conditions
func (a *TektonInstallationAssertion) HasConditions(expected ...toolchainv1alpha1.Condition) *TektonInstallationAssertion {
	err := a.loadTektonInstallationAssertion()
	require.NoError(a.t, err)
	AssertConditionsMatch(a.t, a.tektonInstallation.Status.Conditions, expected...)
	return a
}

func assertThatContainsOwnerReference(t *testing.T, cl client.Client, references []v1.OwnerReference, sub *opsv1alpha1.Subscription) {
	err := cl.Get(context.TODO(), types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}, sub)
	require.NoError(t, err)

	require.Len(t, references, 1)
	assert.Equal(t, sub.GetName(), references[0].Name)
	assert.Equal(t, sub.Kind, references[0].Kind)
	assert.Equal(t, sub.GetUID(), references[0].UID)
	assert.Equal(t, sub.APIVersion, references[0].APIVersion)
	assert.True(t, *references[0].BlockOwnerDeletion)
}
