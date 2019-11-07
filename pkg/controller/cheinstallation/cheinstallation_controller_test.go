package cheinstallation

import (
	"context"
	"errors"
	"fmt"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/k8s"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/olm"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestCheInstallationController(t *testing.T) {
	t.Run("should reconcile with che installation", func(t *testing.T) {
		// given
		cheOperatorNs, cheOg, cheSub := newCheResources()
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)

		request := newReconcileRequest(cheInstallation)

		t.Run("should create ns and requeue", func(t *testing.T) {
			// when
			result, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			assert.True(t, result.Requeue)
			AssertThatNamespace(t, cheOperatorNs, cl).
				Exists().
				HasLabels(toolchain.Labels())

			AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
				DoesNotExist()

			AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
				DoesNotExist()
		})

		t.Run("should create operator group and requeue", func(t *testing.T) {
			// when
			result, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			assert.True(t, result.Requeue)
			AssertThatNamespace(t, cheOperatorNs, cl).
				Exists().
				HasLabels(toolchain.Labels())

			AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
				Exists().
				HasSize(1).
				HasSpec(cheOg.Spec)

			AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
				DoesNotExist()
		})

		t.Run("should create che subscription and requeue", func(t *testing.T) {
			// when
			result, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			assert.True(t, result.Requeue)
			AssertThatNamespace(t, cheOperatorNs, cl).
				Exists().
				HasLabels(toolchain.Labels())

			AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
				Exists().
				HasSize(1).
				HasSpec(cheOg.Spec)

			AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
				Exists().
				HasSpec(cheSub.Spec)
		})

		t.Run("should not requeue", func(t *testing.T) {
			// when
			result, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			assert.False(t, result.Requeue)
			AssertThatNamespace(t, cheOperatorNs, cl).
				Exists().
				HasLabels(toolchain.Labels())

			AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
				Exists().
				HasSize(1).
				HasSpec(cheOg.Spec)

			AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
				Exists().
				HasSpec(cheSub.Spec)

			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
				HasConditions(che.SubscriptionCreated(che.SubscriptionSuccess))
		})

	})

	t.Run("should not reconcile without che installation", func(t *testing.T) {
		// given
		cheOperatorNs, cheOg, cheSub := newCheResources()
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t)

		request := newReconcileRequest(cheInstallation)

		// when
		_, err := r.Reconcile(request)

		// then
		require.NoError(t, err)
		AssertThatNamespace(t, cheOperatorNs, cl).
			DoesNotExist()

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()
	})

	t.Run("update status when failed to get ns", func(t *testing.T) {
		cheOperatorNs, cheOg, cheSub := newCheResources()
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)

		request := newReconcileRequest(cheInstallation)
		errMsg := "something went wrong while getting ns"
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			if _, ok := obj.(*v1.Namespace); ok {
				return errors.New(errMsg)
			}
			return cl.Client.Get(ctx, key, obj)
		}

		// reconcile for che subscription
		_, err := r.Reconcile(request)

		// then
		assert.EqualError(t, err, fmt.Sprintf("failed to create namespace %s: %s", cheOperatorNs, errMsg))

		AssertThatNamespace(t, cheOperatorNs, cl).
			DoesNotExist()

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()

		AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
			HasConditions(che.SubscriptionFailed(errMsg))
	})

	t.Run("should update status when failed to create operator group", func(t *testing.T) {
		cheOperatorNs, cheOg, cheSub := newCheResources()
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)

		request := newReconcileRequest(cheInstallation)

		errMsg := "something went wrong while creating og"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			if _, ok := obj.(*olmv1.OperatorGroup); ok {
				return errors.New(errMsg)
			}
			return cl.Client.Create(ctx, obj, opts...)
		}

		// first reconcile for ns creation
		result, err := r.Reconcile(request)
		require.NoError(t, err)
		assert.True(t, result.Requeue)

		// when
		_, err = r.Reconcile(request)

		// then
		assert.EqualError(t, err, fmt.Sprintf("failed to create operatorgroup in namespace %s: %s", cheOperatorNs, errMsg))

		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(toolchain.Labels())

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()

		AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
			HasConditions(che.SubscriptionFailed(errMsg))
	})

	t.Run("should update status when failed to create che subscription", func(t *testing.T) {
		cheOperatorNs, cheOg, cheSub := newCheResources()
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)

		request := newReconcileRequest(cheInstallation)

		errMsg := "something went wrong while creating che subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			if sub, ok := obj.(*olmv1alpha1.Subscription); ok && sub.Name == che.SubscriptionName {
				return errors.New(errMsg)
			}
			return cl.Client.Create(ctx, obj, opts...)
		}

		// first reconcile for ns creation
		result, err := r.Reconcile(request)
		require.NoError(t, err)
		assert.True(t, result.Requeue)

		// second reconcile for og creation
		result, err = r.Reconcile(request)
		require.NoError(t, err)
		assert.True(t, result.Requeue)

		// when
		_, err = r.Reconcile(request)

		// then
		assert.EqualError(t, err, fmt.Sprintf("failed to create che subscription in namespace %s: %s", cheOperatorNs, errMsg))
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(toolchain.Labels())

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()

		AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
			HasConditions(che.SubscriptionFailed(errMsg))
	})

}

func TestCreateOperatorGroupForChe(t *testing.T) {
	t.Run("create operator group", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// when
		ogCreated, err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, cheInstallation)

		//then
		require.NoError(t, err)

		assert.True(t, ogCreated)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)
	})

	t.Run("should not fail if operator group already exists", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// create for the first time
		ogCreated, err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, cheInstallation)

		require.NoError(t, err)

		assert.True(t, ogCreated)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		// when
		ogCreated, err = r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, cheInstallation)

		// then
		require.NoError(t, err)

		assert.False(t, ogCreated)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)
	})

	t.Run("should fail to create operator group", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		errMsg := "something went wrong while creating operatogrgroup"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// when
		ogCreated, err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, cheInstallation)

		//then
		require.EqualError(t, err, errMsg)

		assert.False(t, ogCreated)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			DoesNotExist()
	})

}

func TestCreateSubscriptionForChe(t *testing.T) {
	t.Run("create subscription", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		cheSub := che.NewSubscription(cheOperatorNs)

		// when
		subCreated, err := r.ensureCheSubscription(testLogger(), cheOperatorNs, cheInstallation)

		// then
		require.NoError(t, err)

		assert.True(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)
	})

	t.Run("should fail to create subscription", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		errMsg := "something went wrong while creating che subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		cheSub := che.NewSubscription(cheOperatorNs)

		// when
		subCreated, err := r.ensureCheSubscription(testLogger(), cheOperatorNs, cheInstallation)

		// then
		require.EqualError(t, err, errMsg)

		assert.False(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()
	})

	t.Run("should not fail if subscription already exists", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		cheSub := che.NewSubscription(cheOperatorNs)

		// create for the first time
		subCreated, err := r.ensureCheSubscription(testLogger(), cheOperatorNs, cheInstallation)
		require.NoError(t, err)

		assert.True(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)

		// when
		subCreated, err = r.ensureCheSubscription(testLogger(), cheOperatorNs, cheInstallation)

		// then
		require.NoError(t, err)

		assert.False(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)
	})

}

func TestCreateNamespaceForChe(t *testing.T) {
	t.Run("should create ns", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)

		// when
		nsCreated, err := r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)

		assert.True(t, nsCreated)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(toolchain.Labels())
	})

	t.Run("should not fail if ns exists", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)

		// create for the first time
		nsCreated, err := r.ensureCheNamespace(testLogger(), cheInstallation)
		require.NoError(t, err)

		assert.True(t, nsCreated)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(toolchain.Labels())

		// when
		nsCreated, err = r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)

		assert.False(t, nsCreated)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(toolchain.Labels())
	})

	t.Run("should fail to create ns", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheInstallation := NewCheInstallation(cheOperatorNs)
		cl, r := configureClient(t, cheInstallation)
		errMsg := "something went wrong while creating ns"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}

		// when
		nsCreated, err := r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.EqualError(t, err, errMsg)

		assert.False(t, nsCreated)
		AssertThatNamespace(t, cheOperatorNs, cl).
			DoesNotExist()
	})

}

func configureClient(t *testing.T, initObjs ...runtime.Object) (*test.FakeClient, *ReconcileCheInstallation) {
	s := apiScheme(t)
	cl := test.NewFakeClient(t, initObjs...)
	reconcileCheInstallation := &ReconcileCheInstallation{scheme: s, client: cl}
	return cl, reconcileCheInstallation
}

func newReconcileRequest(cheInstallation *v1alpha1.CheInstallation) reconcile.Request {
	namespacedName := types.NamespacedName{Namespace: cheInstallation.Namespace, Name: cheInstallation.Name}
	return reconcile.Request{namespacedName}
}

func apiScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}

func testLogger() logr.Logger {
	logger := zap.Logger(true)
	logf.SetLogger(logger)
	return logger
}

func newCheResources() (string, *olmv1.OperatorGroup, *olmv1alpha1.Subscription) {
	cheOperatorNs := GenerateName("che-op")
	return cheOperatorNs, che.NewOperatorGroup(cheOperatorNs), che.NewSubscription(cheOperatorNs)
}
