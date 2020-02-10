package cheinstallation

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	"github.com/codeready-toolchain/toolchain-operator/test"
	. "github.com/codeready-toolchain/toolchain-operator/test/olm"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apiextnv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	// enable logs in tests
	logf.SetLogger(zap.Logger(true))
}

func TestReconcile(t *testing.T) {

	// reconciling on Che Installation resource
	t.Run("Che installation", func(t *testing.T) {

		t.Run("should not reconcile without Che installation", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t)
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatNamespace(t, Namespace, cl).
				DoesNotExist()
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				DoesNotExist()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				DoesNotExist()
		})

		t.Run("should requeue when getting Che installation failed", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cl, r := configureClient(t)
			cl.MockGet = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
				// make sure an error is returned when trying to get the CheInstallation resource
				return fmt.Errorf("fake server error")
			}

			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.Error(t, err)
			assert.Equal(t, "fake server error", err.Error())
		})
	})

	// reconciling on namespace
	t.Run("namespace", func(t *testing.T) {

		t.Run("should create Che namespace", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation)

			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatNamespace(t, Namespace, cl).
				Exists().
				HasLabels(toolchain.Labels())
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				DoesNotExist()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				DoesNotExist()
		})

		t.Run("should update status when failed to get existing namespace", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceActive))
			request := newReconcileRequest(cheInstallation)
			errMsg := "something went wrong while getting ns"
			cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
				if _, ok := obj.(*v1.Namespace); ok {
					return errors.New(errMsg)
				}
				return cl.Client.Get(ctx, key, obj)
			}

			// when
			_, err := r.Reconcile(request)

			// then
			assert.EqualError(t, err, fmt.Sprintf("failed to create namespace %s: %s", Namespace, errMsg))
			AssertThatNamespace(t, Namespace, cl).
				DoesNotExist()
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				DoesNotExist()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
				HasConditions(InstallationFailed(errMsg))
		})

		t.Run("should update status if existing namespace is not active", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceTerminating))
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatNamespace(t, Namespace, cl).Exists()
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).DoesNotExist()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasNoCondition()
		})

	})

	// reconciling on Che OperatorGroup resource
	t.Run("operator group", func(t *testing.T) {

		t.Run("should create Che operator group", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceActive))

			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatNamespace(t, Namespace, cl).
				Exists().
				HasLabels(toolchain.Labels())
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				Exists().
				HasSize(1).
				HasSpec(NewOperatorGroup(cheOperatorNS).Spec)
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				DoesNotExist()
		})

		t.Run("should update status when failed to create operator group", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceActive))
			request := newReconcileRequest(cheInstallation)
			errMsg := "something went wrong while creating og"
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*olmv1.OperatorGroup); ok {
					return errors.New(errMsg)
				}
				return cl.Client.Create(ctx, obj, opts...)
			}

			// when
			_, err := r.Reconcile(request)

			// then
			assert.EqualError(t, err, fmt.Sprintf("failed to create operatorgroup in namespace %s: %s", Namespace, errMsg))
			AssertThatNamespace(t, Namespace, cl).
				Exists().
				HasLabels(toolchain.Labels())
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				DoesNotExist()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				DoesNotExist()

			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
				HasConditions(InstallationFailed(errMsg))
		})

	})

	// reconciling on Che subscription resource
	t.Run("subscription", func(t *testing.T) {

		t.Run("should create Che subscription", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS))
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)

			AssertThatNamespace(t, Namespace, cl).
				Exists().
				HasLabels(toolchain.Labels())
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				Exists().
				HasSize(1).
				HasSpec(NewOperatorGroup(cheOperatorNS).Spec)
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				Exists().
				HasSpec(NewSubscription(cheOperatorNS).Spec)
		})

		t.Run("should update status when failed to create Che subscription", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceActive), NewOperatorGroup(cheOperatorNS))
			request := newReconcileRequest(cheInstallation)
			errMsg := "something went wrong while creating Che subscription"
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				if sub, ok := obj.(*olmv1alpha1.Subscription); ok && sub.Name == SubscriptionName {
					return errors.New(errMsg)
				}
				return cl.Client.Create(ctx, obj, opts...)
			}

			// when
			_, err := r.Reconcile(request)

			// then
			assert.EqualError(t, err, fmt.Sprintf("failed to create Che subscription in namespace %s: %s", Namespace, errMsg))
			AssertThatNamespace(t, Namespace, cl).
				Exists().
				HasLabels(toolchain.Labels())
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
				Exists().
				HasSize(1).
				HasSpec(NewOperatorGroup(cheOperatorNS).Spec)
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
				DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
				HasConditions(InstallationFailed(errMsg))
		})
	})

	// reconciling on checluster resource watcher
	t.Run("checluster watcher", func(t *testing.T) {

		t.Run("should create watcher", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS))
			r.watchCheCluster = func() error {
				return nil
			}
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).Exists()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasNoCondition()
		})

		t.Run("should requeue if CRD does not exist when adding watcher", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS))
			e := meta.NoKindMatchError{GroupKind: schema.GroupKind{Kind: "Foo"}}
			r.watchCheCluster = func() error {
				return &e
			}
			request := newReconcileRequest(cheInstallation)

			// when
			res, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			assert.True(t, res.Requeue)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasNoCondition()
		})

		t.Run("should update Che installation status when adding watcher failed for unknown reason", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS),
				newCustomResourceDefinition(CheClusterCRDName),
			)
			e := errors.New("unexpected error")
			r.watchCheCluster = func() error {
				return e
			}
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.Error(t, err)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasConditions(InstallationFailed("unexpected error"))
		})
	})

	// reconciling on checluster resource
	t.Run("checluster", func(t *testing.T) {

		t.Run("should create checluster", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS))
			r.watchCheCluster = nil // assume the watcher was already created
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).Exists()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasNoCondition()
		})

		t.Run("should update status with existing checluster", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cheCluster.Status.CheClusterRunning = "Installing"
			cheCluster.Status.DbProvisoned = false
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS),
				cheCluster)
			r.watchCheCluster = nil // assume the watcher was already created
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.NoError(t, err)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).Exists()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasConditions(Installing(fmt.Sprintf("Provisioning Database for CheCluster '%s'", cheCluster.Name)))
		})

		t.Run("should update status when failed to create checluster", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS))
			r.watchCheCluster = nil // assume the watcher was already created
			errMsg := "failed to create CheCluster"
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*orgv1.CheCluster); ok {
					return errors.New(errMsg)
				}
				return cl.Client.Create(ctx, obj)
			}
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.Error(t, err)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasConditions(InstallationFailed(errMsg))
		})

		t.Run("should update status when failed to get existing checluster", func(t *testing.T) {
			// given
			cheInstallation := NewInstallation()
			cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
			cheCluster := NewCheCluster(cheOperatorNS)
			cl, r := configureClient(t, cheInstallation,
				newCheNamespace(cheOperatorNS, v1.NamespaceActive),
				NewOperatorGroup(cheOperatorNS),
				NewSubscription(cheOperatorNS))
			r.watchCheCluster = nil // assume the watcher was already created
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*orgv1.CheCluster); ok {
					return apierrors.NewAlreadyExists(schema.GroupResource{}, cheOperatorNS)
				}
				return cl.Client.Create(ctx, obj)
			}
			request := newReconcileRequest(cheInstallation)

			// when
			_, err := r.Reconcile(request)

			// then
			require.Error(t, err)
			AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).Exists()
			AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).Exists()
			AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, cl).DoesNotExist()
			AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).HasConditions(InstallationFailed("checlusters.org.eclipse.che \"codeready-workspaces\" not found"))
		})
	})

	t.Run("should update installation status ready with true upon completion", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cheCluster := NewCheCluster(cheOperatorNS)
		cheCluster.Status.CheClusterRunning = AvailableStatus
		cl, r := configureClient(t, cheInstallation,
			newCheNamespace(cheOperatorNS, v1.NamespaceActive),
			NewOperatorGroup(cheOperatorNS),
			NewSubscription(cheOperatorNS),
			cheCluster)
		request := newReconcileRequest(cheInstallation)

		// when
		_, err := r.Reconcile(request)

		// then
		require.NoError(t, err)

		AssertThatNamespace(t, Namespace, cl).
			Exists().
			HasLabels(toolchain.Labels())
		AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
			Exists().
			HasSize(1).
			HasSpec(NewOperatorGroup(cheOperatorNS).Spec)
		AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
			Exists().
			HasSpec(NewSubscription(cheOperatorNS).Spec)
		AssertThatCheInstallation(t, cheInstallation.Namespace, cheInstallation.Name, cl).
			HasConditions(InstallationSucceeded())
	})

}

func TestCreateOperatorGroupForChe(t *testing.T) {

	t.Run("create operator group", func(t *testing.T) {
		//given
		cheInstallation := NewInstallation()
		cl, r := configureClient(t, cheInstallation)
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace

		// when
		created, err := r.ensureCheOperatorGroup(testLogger(), cheInstallation)

		//then
		require.NoError(t, err)
		assert.True(t, created)
		AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
			Exists().
			HasSize(1).
			HasSpec(NewOperatorGroup(cheOperatorNS).Spec)
	})

	t.Run("should not fail if operator group already exists", func(t *testing.T) {
		//given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cheOg := NewOperatorGroup(cheOperatorNS)
		// OperatorGroup is already exists as provided to fake client
		cl, r := configureClient(t, cheInstallation, cheOg)

		// when
		created, err := r.ensureCheOperatorGroup(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)
		assert.False(t, created)

		AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
			Exists().
			HasSize(1).
			HasSpec(NewOperatorGroup(cheInstallation.Spec.CheOperatorSpec.Namespace).Spec)
	})

	t.Run("should fail to create operator group when error occurs", func(t *testing.T) {
		//given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cl, r := configureClient(t, cheInstallation)
		errMsg := "something went wrong while creating operatogrgroup"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}

		// when
		_, err := r.ensureCheOperatorGroup(testLogger(), cheInstallation)

		//then
		require.EqualError(t, err, errMsg)
		AssertThatOperatorGroup(t, cheOperatorNS, OperatorGroupName, cl).
			DoesNotExist()
	})

}

func TestCreateSubscriptionForChe(t *testing.T) {

	t.Run("create subscription", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cheOperatorGroup := NewOperatorGroup(cheOperatorNS)
		cl, r := configureClient(t, cheInstallation, cheOperatorGroup)

		// when
		created, err := r.ensureCheSubscription(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)
		assert.True(t, created)
		AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
			Exists().
			HasSpec(NewSubscription(cheOperatorNS).Spec)
	})

	t.Run("should fail to create subscription", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cheOperatorGroup := NewOperatorGroup(cheOperatorNS)
		cl, r := configureClient(t, cheInstallation, cheOperatorGroup)
		errMsg := "something went wrong while creating Che subscription"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			// return an error when the object to create is a Subscription
			return errors.New(errMsg)
		}

		// when
		created, err := r.ensureCheSubscription(testLogger(), cheInstallation)

		// then
		require.EqualError(t, err, errMsg)
		assert.False(t, created)
		AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
			DoesNotExist()
	})

	t.Run("should not fail if subscription already exists", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cheSub := NewSubscription(cheOperatorNS)
		// Che Subscription will exists as provided to fake client
		cl, r := configureClient(t, cheInstallation, cheSub)

		// when
		created, err := r.ensureCheSubscription(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)
		assert.False(t, created)
		AssertThatSubscription(t, cheOperatorNS, SubscriptionName, cl).
			Exists().
			HasSpec(NewSubscription(cheOperatorNS).Spec)
	})

}

func TestCreateNamespaceForChe(t *testing.T) {

	t.Run("should create ns", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cl, r := configureClient(t, cheInstallation)

		// when
		requeue, err := r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)
		assert.True(t, requeue)
		AssertThatNamespace(t, Namespace, cl).
			Exists().
			HasLabels(toolchain.Labels())
	})

	t.Run("should not fail if ns exists", func(t *testing.T) {
		//given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceActive))

		// when
		requeue, err := r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)
		assert.False(t, requeue)
		AssertThatNamespace(t, Namespace, cl).
			Exists().
			HasLabels(toolchain.Labels())
	})

	t.Run("should not fail as ns is in termination state", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
		cl, r := configureClient(t, cheInstallation, newCheNamespace(cheOperatorNS, v1.NamespaceTerminating))

		// when
		requeue, err := r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.NoError(t, err)
		assert.True(t, requeue)
		AssertThatNamespace(t, Namespace, cl).Exists()
	})

	t.Run("should fail to create ns", func(t *testing.T) {
		// given
		cheInstallation := NewInstallation()
		cl, r := configureClient(t, cheInstallation)
		errMsg := "something went wrong while creating ns"
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return errors.New(errMsg)
		}

		// when
		requeue, err := r.ensureCheNamespace(testLogger(), cheInstallation)

		// then
		require.EqualError(t, err, errMsg)
		assert.False(t, requeue)
		AssertThatNamespace(t, Namespace, cl).
			DoesNotExist()
	})
}

func TestGetCheClusterStatus(t *testing.T) {
	t.Run("status_unknown_as_nil_input", func(t *testing.T) {
		got := getCheClusterStatus(nil)
		assert.Contains(t, got, fmt.Sprintf("Status is unknown for CheCluster '%s'", CheClusterName))
	})

	t.Run("stauts_unknown_as_blank_status", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			Status: orgv1.CheClusterStatus{},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Status is unknown for CheCluster '%s'", CheClusterName))
	})

	t.Run("db_not_provision", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning: "Unavailable",
				DbProvisoned:      false,
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Provisioning Database for CheCluster '%s'", cluster.Name))
	})

	t.Run("keycloak_not_provision", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning:  "Unavailable",
				DbProvisoned:       true,
				KeycloakProvisoned: false,
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Provisioning Keycloak for CheCluster '%s'", cluster.Name))
	})

	t.Run("openshift_auth_not_provision", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning:         "Unavailable",
				DbProvisoned:              true,
				KeycloakProvisoned:        true,
				OpenShiftoAuthProvisioned: false,
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Provisioning OpenShiftoAuth for CheCluster '%s'", cluster.Name))
	})

	t.Run("devfile_registry_url_not_set", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning:         "Unavailable",
				DbProvisoned:              true,
				KeycloakProvisoned:        true,
				OpenShiftoAuthProvisioned: true,
				DevfileRegistryURL:        "",
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Provisioning DevfileRegistry for CheCluster '%s'", cluster.Name))
	})

	t.Run("plugin_registry_url_not_set", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning:         "Unavailable",
				DbProvisoned:              true,
				KeycloakProvisoned:        true,
				OpenShiftoAuthProvisioned: true,
				DevfileRegistryURL:        "some_url",
				PluginRegistryURL:         "",
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Provisioning PluginRegistry for CheCluster '%s'", cluster.Name))
	})

	t.Run("che_url_not_set", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning:         "Unavailable",
				DbProvisoned:              true,
				KeycloakProvisoned:        true,
				OpenShiftoAuthProvisioned: true,
				DevfileRegistryURL:        "some_url",
				PluginRegistryURL:         "some_url",
				CheURL:                    "",
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("Provisioning CheServer for CheCluster '%s'", cluster.Name))
	})

	t.Run("status_", func(t *testing.T) {
		cluster := &orgv1.CheCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeready-workspaces",
			},
			Status: orgv1.CheClusterStatus{
				CheClusterRunning:         "Unavailable",
				DbProvisoned:              true,
				KeycloakProvisoned:        true,
				OpenShiftoAuthProvisioned: true,
				DevfileRegistryURL:        "some_url",
				PluginRegistryURL:         "some_url",
				CheURL:                    "some_url",
			},
		}
		got := getCheClusterStatus(cluster)
		assert.Contains(t, got, fmt.Sprintf("CheCluster running status is '%s' for CheCluster '%s'", cluster.Status.CheClusterRunning, cluster.Name))
	})
}

func TestEnsureWatchCheCluster(t *testing.T) {

	t.Run("add_watch_ok", func(t *testing.T) {
		cl, r := configureClient(t)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return nil
		}
		r.watchCheCluster = func() error {
			return nil
		}

		// test
		requeue, err := r.ensureWatchCheCluster()

		require.NoError(t, err)
		assert.False(t, requeue)
		assert.Nil(t, r.watchCheCluster)
	})

	t.Run("add_watch_requeue_as_kind_not_found", func(t *testing.T) {
		cl, r := configureClient(t)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return nil
		}
		r.watchCheCluster = func() error {
			return &meta.NoKindMatchError{GroupKind: schema.GroupKind{Kind: "Foo"}}
		}

		// test
		requeue, err := r.ensureWatchCheCluster()

		require.NoError(t, err)
		assert.True(t, requeue)
	})

	t.Run("add_watch_failed_with_unknown_error", func(t *testing.T) {
		cl, r := configureClient(t)
		cl.MockGet = func(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
			return nil
		}
		errMsg := "unknown"
		r.watchCheCluster = func() error {
			return fmt.Errorf(errMsg)
		}

		// test
		requeue, err := r.ensureWatchCheCluster()

		require.Error(t, err)
		assert.EqualError(t, err, errMsg)
		assert.False(t, requeue)
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
	return reconcile.Request{NamespacedName: namespacedName}
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

func newCheNamespace(ns string, nsPhase v1.NamespacePhase) *v1.Namespace {
	cheNs := NewNamespace(ns)
	cheNs.Status.Phase = nsPhase
	return cheNs
}

func newCustomResourceDefinition(name string) *apiextnv1beta1.CustomResourceDefinition {
	return &apiextnv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
