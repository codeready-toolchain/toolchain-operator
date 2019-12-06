package olm

import (
	"context"
	testwait "github.com/codeready-toolchain/toolchain-operator/test/wait"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CheClusterAssertion struct {
	t              *testing.T
	client         client.Reader
	namespacedName types.NamespacedName
	cheCluster     *orgv1.CheCluster
}

func AssertThatCheCluster(t *testing.T, ns, name string, client client.Reader) *CheClusterAssertion {
	return &CheClusterAssertion{
		t:              t,
		client:         client,
		namespacedName: types.NamespacedName{Namespace: ns, Name: name},
	}
}

func (a *CheClusterAssertion) Exists() *CheClusterAssertion {
	err := a.loadCheClusterAssertion()
	require.NoError(a.t, err)
	return a
}

func (a *CheClusterAssertion) HasRunningStatus(want string) *CheClusterAssertion {
	a.Exists()
	assert.Equal(a.t, want, a.cheCluster.Status.CheClusterRunning)
	return a
}

func (a *CheClusterAssertion) DoesNotExist() *CheClusterAssertion {
	err := testwait.PollOnceOrUntilCondition(func() (done bool, err error) {
		err = a.loadCheClusterAssertion()
		if err != nil {
			if errors.IsNotFound(err) {
				a.t.Logf("CheCluster deleted from namespace")
				return true, err
			}
			return false, err
		}
		a.t.Logf("waiting for CheCluster '%s' to be deleted from namespace '%s'", a.cheCluster.Name, a.cheCluster.Namespace)
		return false, nil
	})

	require.Error(a.t, err)
	assert.IsType(a.t, metav1.StatusReasonNotFound, errors.ReasonForError(err))
	return a
}

func (a *CheClusterAssertion) loadCheClusterAssertion() error {
	if a.cheCluster != nil {
		return nil
	}
	cheCluster := &orgv1.CheCluster{}
	err := a.client.Get(context.TODO(), a.namespacedName, cheCluster)
	a.cheCluster = cheCluster
	return err
}
