package olm

import (
	"context"
	"github.com/codeready-toolchain/toolchain-operator/pkg/utils/che"
	testwait "github.com/codeready-toolchain/toolchain-operator/test/wait"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type OperatorGroupAssertion struct {
	ogList         []olmv1.OperatorGroup
	client         client.Reader
	namespacedName types.NamespacedName
	t              *testing.T
}

func (a *OperatorGroupAssertion) loadOperatorGroupAssertion() error {
	if a.ogList != nil {
		return nil
	}
	ogList := &olmv1.OperatorGroupList{}
	err := a.client.List(context.TODO(), ogList, client.InNamespace(a.namespacedName.Namespace), client.MatchingLabels(che.Labels()))

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
	err := wait.Poll(testwait.RetryInterval, testwait.Timeout, func() (done bool, err error) {
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
	assert.EqualValues(a.t, a.ogList[0].Spec, ogSpec)
	return a
}
