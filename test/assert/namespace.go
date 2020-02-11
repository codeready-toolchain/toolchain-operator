package assert

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceAssertion struct {
	namespace      *v1.Namespace
	client         client.Reader
	namespacedName types.NamespacedName
	t              *testing.T
}

func (a *NamespaceAssertion) loadNamespaceAssertion() error {
	ns := &v1.Namespace{}
	err := a.client.Get(context.TODO(), a.namespacedName, ns)
	a.namespace = ns
	return err
}

func AssertThatNamespace(t *testing.T, name string, client client.Reader) *NamespaceAssertion {
	return &NamespaceAssertion{
		client:         client,
		namespacedName: types.NamespacedName{Name: name},
		t:              t,
	}
}

func (a *NamespaceAssertion) DoesNotExist() *NamespaceAssertion {
	err := PollOnceOrUntilCondition(func() (done bool, err error) {
		err = a.loadNamespaceAssertion()
		if err != nil {
			if errors.IsNotFound(err) {
				a.t.Logf("Namespace deleted")
				return true, err
			}
			return false, err
		}
		a.t.Logf("waiting for namespace '%s', status: '%s' to be deleted", a.namespace.Name, a.namespace.Status)
		return false, nil
	})
	require.Error(a.t, err)
	assert.IsType(a.t, metav1.StatusReasonNotFound, errors.ReasonForError(err))
	return a
}

func (a *NamespaceAssertion) Exists() *NamespaceAssertion {
	err := a.loadNamespaceAssertion()
	require.NoError(a.t, err)
	return a
}

func (a *NamespaceAssertion) HasLabels(labels map[string]string) *NamespaceAssertion {
	err := a.loadNamespaceAssertion()
	require.NoError(a.t, err)
	assert.EqualValues(a.t, a.namespace.Labels, labels)
	return a
}
