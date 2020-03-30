package tektoninstallation

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
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
		cl, r := configureClient(t, tektonInstallation)
		request := newReconcileRequest(tektonInstallation)

		t.Run("should create tekton subscription and requeue", func(t *testing.T) {
			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(InstallationInstalling("created tekton subscription"))

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)
		})

		t.Run("should not requeue", func(t *testing.T) {
			cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
				if _, ok := obj.(*config.Config); ok {
					installedCode := []config.ConfigCondition{
						config.ConfigCondition{
							Code: "applied-addons",
						},
						config.ConfigCondition{
							Code: config.InstalledStatus,
						},
						config.ConfigCondition{
							Code: "validated-pipeline",
						},
					}
					obj = NewTektonConfig(installedCode...)
					return nil
				}

				return cl.Client.Get(ctx, key, obj)
			}

			// when
			result, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			assert.False(t, result.Requeue)
			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)

			AssertThatTektonInstallation(t, tektonInstallation.Namespace, tektonInstallation.Name, cl).
				HasConditions(InstallationUnknown())
		})

	})

	// reconciling on tektonconfig resource watcher
	t.Run("tektonconfig watcher", func(t *testing.T) {

		t.Run("installed tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			installedCode := []config.ConfigCondition{
				config.ConfigCondition{
					Code: "applied-addons",
				},
				config.ConfigCondition{
					Code: config.InstalledStatus,
				},
				config.ConfigCondition{
					Code: "validated-pipeline",
				},
			}
			tektonConfig := NewTektonConfig(installedCode...)
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
				HasConditions(InstallationSucceeded())
		})

		t.Run("installing tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			installingCode := config.ConfigCondition{
				Code: config.InstallingStatus,
			}
			tektonConfig := NewTektonConfig(installingCode)
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
				HasConditions(InstallationInstalling("tekton installation installing"))
		})

		t.Run("error with tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			errorCode := config.ConfigCondition{
				Code: config.ErrorStatus,
			}
			tektonConfig := NewTektonConfig(errorCode)
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
				HasConditions(InstallationFailed("tekton installation failed with error: "))
		})

		t.Run("unknown status with tekton installation", func(t *testing.T) {
			// given
			tektonInstallation := NewInstallation()
			unknownCode := config.ConfigCondition{
				Code: "applied-addons",
			}
			tektonConfig := NewTektonConfig(unknownCode)
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
				HasConditions(InstallationUnknown())
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

// NewTektonConfig returns a new TektonConfig with the given conditions
func NewTektonConfig(conditions ...config.ConfigCondition) *config.Config {
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
			Conditions: conditions,
		},
	}
}
