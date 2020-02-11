package assert

import (
	"context"
	"testing"

	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"

	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatorGroupAssertion struct {
	ogList         []olmv1.OperatorGroup
	client         client.Reader
	namespacedName types.NamespacedName
	t              *testing.T
}

func (a *OperatorGroupAssertion) loadOperatorGroupAssertion() error {
	ogList := &olmv1.OperatorGroupList{}
	err := a.client.List(context.TODO(), ogList, client.InNamespace(a.namespacedName.Namespace), client.MatchingLabels(toolchain.Labels()))

	a.ogList = ogList.Items
	return err
}

func AssertThatOperatorGroup(t *testing.T, ns, name string, client client.Reader) *OperatorGroupAssertion {
	return &OperatorGroupAssertion{
		client:         client,
		namespacedName: types.NamespacedName{Namespace: ns, Name: name},
		t:              t,
	}
}

func (a *OperatorGroupAssertion) DoesNotExist() *OperatorGroupAssertion {
	err := PollOnceOrUntilCondition(func() (done bool, err error) {
		err = a.loadOperatorGroupAssertion()
		if len(a.ogList) == 0 {
			a.t.Logf("operatorgroup deleted")
			return true, nil
		}
		a.t.Logf("waiting for operatorgroup '%v' to be deleted from namespace '%s'", a.namespacedName.Name, a.namespacedName.Namespace)
		return false, nil
	})

	require.NoError(a.t, err)
	assert.Len(a.t, a.ogList, 0)
	return a
}

func (a *OperatorGroupAssertion) Exists() *OperatorGroupAssertion {
	err := a.loadOperatorGroupAssertion()
	require.NoError(a.t, err)
	return a
}

func (a *OperatorGroupAssertion) HasSize(size int) *OperatorGroupAssertion {
	err := a.loadOperatorGroupAssertion()
	require.NoError(a.t, err)
	assert.Len(a.t, a.ogList, size)
	return a
}

func (a *OperatorGroupAssertion) HasSpec(ogSpec olmv1.OperatorGroupSpec) *OperatorGroupAssertion {
	err := a.loadOperatorGroupAssertion()
	require.NoError(a.t, err)
	require.Len(a.t, a.ogList, 1)
	assert.EqualValues(a.t, a.ogList[0].Spec, ogSpec)
	return a
}
