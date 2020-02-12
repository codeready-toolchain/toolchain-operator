package assert

import (
	"context"
	"testing"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SubscriptionAssertion struct {
	subscription   *olmv1alpha1.Subscription
	client         client.Reader
	namespacedName types.NamespacedName
	t              *testing.T
}

func (a *SubscriptionAssertion) loadSubscriptionAssertion() error {
	subscription := &olmv1alpha1.Subscription{}
	err := a.client.Get(context.TODO(), a.namespacedName, subscription)
	a.subscription = subscription
	return err
}

func AssertThatSubscription(t *testing.T, ns, name string, client client.Reader) *SubscriptionAssertion {
	return &SubscriptionAssertion{
		client:         client,
		namespacedName: types.NamespacedName{Namespace: ns, Name: name},
		t:              t,
	}
}

func (a *SubscriptionAssertion) DoesNotExist() *SubscriptionAssertion {
	err := PollOnceOrUntilCondition(func() (done bool, err error) {
		err = a.loadSubscriptionAssertion()
		if err != nil {
			if errors.IsNotFound(err) {
				a.t.Logf("Subscription deleted from namespace")
				return true, err
			}
			return false, err
		}
		a.t.Logf("waiting for subscription '%s' to be deleted from namespace '%s'", a.subscription.Name, a.subscription.Namespace)
		return false, nil
	})

	require.Error(a.t, err)
	assert.IsType(a.t, metav1.StatusReasonNotFound, errors.ReasonForError(err))
	return a
}

func (a *SubscriptionAssertion) Exists() *SubscriptionAssertion {
	err := a.loadSubscriptionAssertion()
	require.NoError(a.t, err)
	return a
}

func (a *SubscriptionAssertion) HasSpec(subscriptionSpec *olmv1alpha1.SubscriptionSpec) *SubscriptionAssertion {
	err := a.loadSubscriptionAssertion()
	require.NoError(a.t, err)
	assert.EqualValues(a.t, a.subscription.Spec, subscriptionSpec)
	return a
}
