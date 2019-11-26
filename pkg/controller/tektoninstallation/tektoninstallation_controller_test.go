package tektoninstallation

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/tekton"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/olm"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestTektonInstallationController(t *testing.T) {
	t.Run("should reconcile with tekton installation", func(t *testing.T) {
		// given
		tektonSub := tekton.NewSubscription(tekton.SubscriptionNamespace)
		tektonInstallation := NewTektonInstallation()
		cl, r := configureClient(t, tektonInstallation)
		request := newReconcileRequest(tektonInstallation)

		t.Run("should create tekton subscription and requeue", func(t *testing.T) {
			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(tekton.SubscriptionCreated())

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)
		})

		t.Run("should not requeue", func(t *testing.T) {
			// when
			result, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			assert.False(t, result.Requeue)
			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)

			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(tekton.SubscriptionCreated())
		})

	})
}

func TestFailingStatusForTektonInstallation(t *testing.T) {
	// given
	tektonSub := tekton.NewSubscription(tekton.SubscriptionNamespace)

	tektonInstallation := NewTektonInstallation()
	cl, r := configureClient(t, tektonInstallation)

	request := newReconcileRequest(tektonInstallation)

	errMsg := "something went wrong while creating tekton subscription"
	cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		if _, ok := obj.(*olmv1alpha1.Subscription); ok {
			return errors.New(errMsg)
		}
		return cl.Client.Create(ctx, obj, opts...)
	}
	// when
	_, err := r.Reconcile(request)

	// then
	assert.EqualError(t, err, fmt.Sprintf("failed to create tekton subscription in namespace %s: %s", tektonSub.Namespace, errMsg))

	AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
		DoesNotExist()

	AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
		HasConditions(tekton.SubscriptionFailed(errMsg))
}

func TestCreateSubscriptionForTekton(t *testing.T) {
	testLogger := zap.Logger(true)
	logf.SetLogger(testLogger)

	t.Run("create subscription", func(t *testing.T) {
		// given
		tektonSubNs := GenerateName("tekton-op")
		tektonInstallation := NewTektonInstallation()
		cl, r := configureClient(t, tektonInstallation)
		tektonSub := tekton.NewSubscription(tektonSubNs)

		// when
		err := r.ensureTektonSubscription(testLogger, tektonInstallation, tektonSubNs)

		// then
		require.NoError(t, err)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)
	})

	t.Run("should fail to create subscription", func(t *testing.T) {
		// given
		tektonSubNs := GenerateName("tekton-op")
		tektonInstallation := NewTektonInstallation()
		cl, r := configureClient(t, tektonInstallation)
		errMsg := "something went wrong while creating tekton subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		tektonSub := tekton.NewSubscription(tektonSubNs)

		// when
		err := r.ensureTektonSubscription(testLogger, tektonInstallation, tektonSubNs)

		// then
		require.EqualError(t, err, errMsg)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			DoesNotExist()
	})

	t.Run("should not fail if subscription already exists", func(t *testing.T) {
		// given
		tektonSubNs := GenerateName("tekton-op")
		tektonInstallation := NewTektonInstallation()
		tektonSub := tekton.NewSubscription(tektonSubNs)
		cl, r := configureClient(t, tektonInstallation, tektonSub)

		// when
		err := r.ensureTektonSubscription(testLogger, tektonInstallation, tektonSubNs)

		// then
		require.NoError(t, err)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)
	})
}

func configureClient(t *testing.T, initObjs ...runtime.Object) (*test.FakeClient, *ReconcileTektonInstallation) {
	s := apiScheme(t)
	cl := test.NewFakeClient(t, initObjs...)
	reconcileTektonInstallation := &ReconcileTektonInstallation{scheme: s, client: cl}
	return cl, reconcileTektonInstallation
}

func apiScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}

func newReconcileRequest(tektonInstallation *v1alpha1.TektonInstallation) reconcile.Request {
	namespacedName := types.NamespacedName{Namespace: tektonInstallation.Namespace, Name: tektonInstallation.Name}
	return reconcile.Request{NamespacedName: namespacedName}
}
