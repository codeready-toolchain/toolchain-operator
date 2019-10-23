package installconfig

import (
	"context"
	"errors"
	"fmt"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	"github.com/codeready-toolchain/toolchain-operator/pkg/tekton"
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

func TestInstallConfigController(t *testing.T) {
	t.Run("should reconcile with installconfig", func(t *testing.T) {
		// given
		cheOperatorNs, cheOg, cheSub := newCheResources()
		tektonSub := tekton.NewSubscription(tekton.SubscriptionNamespace)
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)

		request := newReconcileRequest(installConfig)

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

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
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

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
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

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				DoesNotExist()

		})

		t.Run("should create tekton subscription and requeue", func(t *testing.T) {
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

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)

			AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
				HasConditions(tekton.SubscriptionCreated(tekton.SubscriptionSuccess), che.SubscriptionCreated(che.SubscriptionSuccess))
		})

	})

	t.Run("should not reconcile without installconfig", func(t *testing.T) {
		// given
		cheOperatorNs, cheOg, cheSub := newCheResources()
		tektonSub := tekton.NewSubscription(tekton.SubscriptionNamespace)

		cl, r := configureClient(t, cheOperatorNs)

		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		request := newReconcileRequest(installConfig)

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

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			DoesNotExist()
	})

	t.Run("should update status failed when something bad happens", func(t *testing.T) {
		tektonSub := tekton.NewSubscription(tekton.SubscriptionNamespace)

		t.Run("update status when failed to get ns", func(t *testing.T) {
			cheOperatorNs, cheOg, cheSub := newCheResources()
			installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
			cl, r := configureClient(t, cheOperatorNs, installConfig)

			request := newReconcileRequest(installConfig)
			errMsg := "something went wrong while getting ns"
			cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
				if _, ok := obj.(*v1.Namespace); ok {
					return errors.New(errMsg)
				}
				return cl.Client.Get(ctx, key, obj)
			}

			// reconcile for che subscription
			result, err := r.Reconcile(request)
			require.NoError(t, err)
			assert.True(t, result.Requeue)

			// reconcile for tekton subscription
			_, err = r.Reconcile(request)

			// then
			assert.EqualError(t, err, fmt.Sprintf("failed to create namespace %s: %s", cheOperatorNs, errMsg))

			AssertThatNamespace(t, cheOperatorNs, cl).
				DoesNotExist()

			AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
				DoesNotExist()

			AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
				DoesNotExist()

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)

			AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
				HasConditions(tekton.SubscriptionCreated(tekton.SubscriptionSuccess), che.SubscriptionFailed(errMsg))
		})

		t.Run("should update status when failed to create operator group", func(t *testing.T) {
			cheOperatorNs, cheOg, cheSub := newCheResources()
			installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
			cl, r := configureClient(t, cheOperatorNs, installConfig)

			request := newReconcileRequest(installConfig)

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

			// reconcile for tekton subscription
			result, err = r.Reconcile(request)
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

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)

			AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
				HasConditions(che.SubscriptionFailed(errMsg), tekton.SubscriptionCreated(tekton.SubscriptionSuccess))
		})

		t.Run("should update status when failed to create che subscription", func(t *testing.T) {
			cheOperatorNs, cheOg, cheSub := newCheResources()
			installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
			cl, r := configureClient(t, cheOperatorNs, installConfig)

			request := newReconcileRequest(installConfig)

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

			// reconcile for tekton subscription
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

			AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
				Exists().
				HasSpec(tektonSub.Spec)

			AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
				HasConditions(che.SubscriptionFailed(errMsg), tekton.SubscriptionCreated(tekton.SubscriptionSuccess))
		})

		t.Run("should update status when failed to create tekton subscription", func(t *testing.T) {
			cheOperatorNs, cheOg, cheSub := newCheResources()
			installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
			cl, r := configureClient(t, cheOperatorNs, installConfig)

			request := newReconcileRequest(installConfig)

			errMsg := "something went wrong while creating tekton subscription"
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				if sub, ok := obj.(*olmv1alpha1.Subscription); ok && sub.Namespace == tekton.SubscriptionNamespace {
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

			// third reconcile for che subscription creation
			result, err = r.Reconcile(request)
			require.NoError(t, err)
			assert.True(t, result.Requeue)

			// when
			_, err = r.Reconcile(request)

			// then
			assert.EqualError(t, err, fmt.Sprintf("failed to create tekton subscription in namespace %s: %s", tektonSub.Namespace, errMsg))
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

			AssertThatSubscription(t, tektonSub.Namespace, cheSub.Name, cl).
				DoesNotExist()

			AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
				HasConditions(che.SubscriptionCreated(che.SubscriptionSuccess), tekton.SubscriptionFailed(errMsg))
		})

		t.Run("should update status when failed to create che and tekton subscription", func(t *testing.T) {
			cheOperatorNs, cheOg, cheSub := newCheResources()
			installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
			cl, r := configureClient(t, cheOperatorNs, installConfig)

			request := newReconcileRequest(installConfig)

			errMsg := "something went wrong while creating tekton subscription"
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*olmv1alpha1.Subscription); ok {
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
			assert.EqualError(t, err, fmt.Sprintf("[failed to create che subscription in namespace %s: %s, failed to create tekton subscription in namespace %s: %s]", cheSub.Namespace, errMsg, tektonSub.Namespace, errMsg))
			AssertThatNamespace(t, cheOperatorNs, cl).
				Exists().
				HasLabels(toolchain.Labels())

			AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
				Exists().
				HasSize(1).
				HasSpec(cheOg.Spec)

			AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
				DoesNotExist()

			AssertThatSubscription(t, tektonSub.Namespace, cheSub.Name, cl).
				DoesNotExist()

			AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
				HasConditions(che.SubscriptionFailed(errMsg), tekton.SubscriptionFailed(errMsg))
		})
	})

}

func TestCreateOperatorGroupForChe(t *testing.T) {
	t.Run("create operator group", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// when
		ogCreated, err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)

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
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// create for the first time
		ogCreated, err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)

		require.NoError(t, err)

		assert.True(t, ogCreated)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		// when
		ogCreated, err = r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)

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
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		errMsg := "something went wrong while creating operatogrgroup"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// when
		ogCreated, err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)

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
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheSub := che.NewSubscription(cheOperatorNs)

		// when
		subCreated, err := r.ensureCheSubscription(testLogger(), cheOperatorNs, installConfig)

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
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		errMsg := "something went wrong while creating che subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		cheSub := che.NewSubscription(cheOperatorNs)

		// when
		subCreated, err := r.ensureCheSubscription(testLogger(), cheOperatorNs, installConfig)

		// then
		require.EqualError(t, err, errMsg)

		assert.False(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()
	})

	t.Run("should not fail if subscription already exists", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheSub := che.NewSubscription(cheOperatorNs)

		// create for the first time
		subCreated, err := r.ensureCheSubscription(testLogger(), cheOperatorNs, installConfig)
		require.NoError(t, err)

		assert.True(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)

		// when
		subCreated, err = r.ensureCheSubscription(testLogger(), cheOperatorNs, installConfig)

		// then
		require.NoError(t, err)

		assert.False(t, subCreated)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)
	})

}

func TestCreateSubscriptionForTekton(t *testing.T) {
	t.Run("create subscription", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		tektonSubNs := GenerateName("tekton-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		tektonSub := tekton.NewSubscription(tektonSubNs)

		// when
		subCreated, err := r.ensureTektonSubscription(testLogger(), tektonSubNs, installConfig)

		// then
		require.NoError(t, err)

		assert.True(t, subCreated)
		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)
	})

	t.Run("should fail to create subscription", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		tektonSubNs := GenerateName("tekton-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		errMsg := "something went wrong while creating tekton subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}
		tektonSub := tekton.NewSubscription(cheOperatorNs)

		// when
		subCreated, err := r.ensureTektonSubscription(testLogger(), tektonSubNs, installConfig)

		// then
		require.EqualError(t, err, errMsg)

		assert.False(t, subCreated)
		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			DoesNotExist()
	})

	t.Run("should not fail if subscription already exists", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		tektonSubNs := GenerateName("tekton-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		tektonSub := tekton.NewSubscription(tektonSubNs)

		// create for the first time
		subCreated, err := r.ensureTektonSubscription(testLogger(), tektonSubNs, installConfig)
		require.NoError(t, err)

		assert.True(t, subCreated)
		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)

		// when
		subCreated, err = r.ensureTektonSubscription(testLogger(), tektonSubNs, installConfig)

		// then
		require.NoError(t, err)

		assert.False(t, subCreated)
		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, cl).
			Exists().
			HasSpec(tektonSub.Spec)
	})
}

func TestCreateNamespaceForChe(t *testing.T) {
	t.Run("should create ns", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)

		// when
		nsCreated, err := r.ensureCheNamespace(testLogger(), installConfig)

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
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)

		// create for the first time
		nsCreated, err := r.ensureCheNamespace(testLogger(), installConfig)
		require.NoError(t, err)

		assert.True(t, nsCreated)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(toolchain.Labels())

		// when
		nsCreated, err = r.ensureCheNamespace(testLogger(), installConfig)

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
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		errMsg := "something went wrong while creating ns"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}

		// when
		nsCreated, err := r.ensureCheNamespace(testLogger(), installConfig)

		// then
		require.EqualError(t, err, errMsg)

		assert.False(t, nsCreated)
		AssertThatNamespace(t, cheOperatorNs, cl).
			DoesNotExist()
	})

}

func configureClient(t *testing.T, cheOperatorNs string, initObjs ...runtime.Object) (*test.FakeClient, *ReconcileInstallConfig) {
	s := apiScheme(t)
	cl := test.NewFakeClient(t, initObjs...)
	reconcileInstallConfig := &ReconcileInstallConfig{scheme: s, client: cl}
	return cl, reconcileInstallConfig
}

func newReconcileRequest(installConfig *v1alpha1.InstallConfig) reconcile.Request {
	namespacedName := types.NamespacedName{Namespace: installConfig.Namespace, Name: installConfig.Name}
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
