package tektoninstallation

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/test"
	. "github.com/codeready-toolchain/toolchain-operator/test/assert"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
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
		tektonSub := NewSubscription(SubscriptionNamespace)
		tektonInstallation := NewInstallation()
		tektonConfig := newTektonConfig("applied-addons", "validated-pipeline")
		cl, r := configureClient(t, tektonInstallation, tektonConfig)
		request := newReconcileRequest(tektonInstallation)

		t.Run("should create tekton subscription and requeue", func(t *testing.T) {
			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(Installing("created tekton subscription"))

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
				HasConditions(Unknown())
		})

	})

	// reconciling on tektonconfig resource watcher
	t.Run("tektonconfig watcher", func(t *testing.T) {

		t.Run("installed tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			tektonConfig := newTektonConfig("applied-addons", config.InstalledStatus, "validated-pipeline")
			cl, r := configureClient(t, tektonInstallation,
				NewSubscription(SubscriptionNamespace),
				tektonConfig)
			r.watchTektonConfig = func() error {
				return nil
			}
			request := newReconcileRequest(tektonInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(InstallationSucceeded())
		})

		t.Run("installing tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			tektonConfig := newTektonConfig(config.InstallingStatus)
			cl, r := configureClient(t, tektonInstallation,
				NewSubscription(SubscriptionNamespace),
				tektonConfig)
			r.watchTektonConfig = func() error {
				return nil
			}
			request := newReconcileRequest(tektonInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatSubscription(t, SubscriptionNamespace, SubscriptionName, cl).Exists()
			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(Installing("tektoninstallation test"))
		})

		t.Run("error with tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			tektonConfig := newTektonConfig(config.ErrorStatus)
			cl, r := configureClient(t, tektonInstallation,
				NewSubscription(SubscriptionNamespace),
				tektonConfig)
			r.watchTektonConfig = func() error {
				return nil
			}
			request := newReconcileRequest(tektonInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatSubscription(t, SubscriptionNamespace, SubscriptionName, cl).Exists()
			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(InstallationFailed("tektoninstallation test"))
		})

		t.Run("unknown status with tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			tektonConfig := newTektonConfig("applied-addons")
			cl, r := configureClient(t, tektonInstallation,
				NewSubscription(SubscriptionNamespace),
				tektonConfig)
			r.watchTektonConfig = func() error {
				return nil
			}
			request := newReconcileRequest(tektonInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatSubscription(t, SubscriptionNamespace, SubscriptionName, cl).Exists()
			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(Unknown())
		})
	})
}

func TestFailingStatusForTektonInstallation(t *testing.T) {
	// given
	tektonSub := NewSubscription(SubscriptionNamespace)

	tektonInstallation := NewInstallation()
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
		HasConditions(InstallationFailed(errMsg))
}

func TestCreateSubscriptionForTekton(t *testing.T) {
	testLogger := zap.Logger(true)
	logf.SetLogger(testLogger)

	t.Run("create subscription", func(t *testing.T) {
		// given
		tektonSubNs := generateName("tekton-op")
		tektonInstallation := NewInstallation()
		cl, r := configureClient(t, tektonInstallation)
		tektonSub := NewSubscription(tektonSubNs)

		// when
		created, err := r.ensureTektonSubscription(testLogger, tektonInstallation, tektonSubNs)

		// then
		require.NoError(t, err)
		require.True(t, created)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)
	})

	t.Run("should fail to create subscription", func(t *testing.T) {
		// given
		tektonSubNs := generateName("tekton-op")
		tektonInstallation := NewInstallation()
		cl, r := configureClient(t, tektonInstallation)
		errMsg := "something went wrong while creating tekton subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		tektonSub := NewSubscription(tektonSubNs)

		// when
		created, err := r.ensureTektonSubscription(testLogger, tektonInstallation, tektonSubNs)

		// then
		require.EqualError(t, err, errMsg)
		require.False(t, created)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			DoesNotExist()
	})

	t.Run("should not fail if subscription already exists", func(t *testing.T) {
		// given
		tektonSubNs := generateName("tekton-op")
		tektonInstallation := NewInstallation()
		tektonSub := NewSubscription(tektonSubNs)
		cl, r := configureClient(t, tektonInstallation, tektonSub)

		// when
		created, err := r.ensureTektonSubscription(testLogger, tektonInstallation, tektonSubNs)

		// then
		require.NoError(t, err)
		require.False(t, created)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)
	})
}

func TestEnsureWatchTektonCluster(t *testing.T) {

	t.Run("add watch ok", func(t *testing.T) {
		cl, r := configureClient(t)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return nil
		}
		r.watchTektonConfig = func() error {
			return nil
		}

		// test
		requeue, err := r.ensureWatchTektonConfig()

		require.NoError(t, err)
		assert.False(t, requeue)
		assert.Nil(t, r.watchTektonConfig)
	})

	t.Run("add watch requeue as kind not found", func(t *testing.T) {
		cl, r := configureClient(t)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return &meta.NoKindMatchError{}
		}
		r.watchTektonConfig = func() error {
			return nil
		}
		// test
		requeue, err := r.ensureWatchTektonConfig()

		require.NoError(t, err)
		assert.True(t, requeue)
	})

	t.Run("add watch failed with unknown error", func(t *testing.T) {
		cl, r := configureClient(t)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return nil
		}
		errMsg := "unknown"
		r.watchTektonConfig = func() error {
			return fmt.Errorf(errMsg)
		}

		// test
		requeue, err := r.ensureWatchTektonConfig()

		require.Error(t, err)
		assert.EqualError(t, err, errMsg)
		assert.False(t, requeue)
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

// generateName return the given name with a suffix based on the current time (UnixNano)
func generateName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// newTektonConfig returns a new TektonConfig with the given conditions
func newTektonConfig(conditions ...config.InstallStatus) *config.Config {
	var codes []config.ConfigCondition
	for _, code := range conditions {
		condition := config.ConfigCondition{
			Code:    config.InstallStatus(code),
			Details: "tektoninstallation test",
		}
		codes = append(codes, condition)
	}

	return &config.Config{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:   TektonConfigName,
			Labels: toolchain.Labels(),
		},
		Spec: config.ConfigSpec{
			TargetNamespace: "",
		},
		Status: config.ConfigStatus{
			Conditions: codes,
		},
	}
}
