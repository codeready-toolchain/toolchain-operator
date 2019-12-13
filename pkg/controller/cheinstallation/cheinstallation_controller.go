package cheinstallation

import (
	"context"
	"fmt"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	errs "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apiextnv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_cheinstallation")

// Add creates a new CheInstallation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileCheInstallation {
	return &ReconcileCheInstallation{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileCheInstallation) error {
	// Create a new controller
	c, err := controller.New("cheinstallation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CheInstallation
	err = c.Watch(&source.Kind{Type: &v1alpha1.CheInstallation{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource
	enqueueRequestForOwner := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.CheInstallation{},
	}

	if err := c.Watch(&source.Kind{Type: &v1.Namespace{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &olmv1.OperatorGroup{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &olmv1alpha1.Subscription{}}, enqueueRequestForOwner); err != nil {
		return err
	}

	watchCheCluster := func() error {
		return c.Watch(&source.Kind{Type: &orgv1.CheCluster{}}, enqueueRequestForOwner)
	}

	err = watchCheCluster()
	if err != nil {
		if !meta.IsNoMatchError(err) { // ignore NoKindMatchError
			return err
		}
		r.watchCheCluster = watchCheCluster
	}
	return nil
}

// blank assignment to verify that ReconcileCheInstallation implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCheInstallation{}

// ReconcileCheInstallation reconciles a CheInstallation object
type ReconcileCheInstallation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	watchCheCluster func() error
}

// Reconcile reads that state of the cluster for a CheInstallation object and makes changes based on the state read
// and what is in the CheInstallation.Spec
func (r *ReconcileCheInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CheInstallation")

	cheInstallation := &v1alpha1.CheInstallation{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cheInstallation)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("CheInstallation not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if requeue, err := r.ensureCheNamespace(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if requeue {
		return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	}

	if created, err := r.ensureCheOperatorGroup(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create operatorgroup in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if created {
		return reconcile.Result{}, nil
	}

	if created, err := r.ensureCheSubscription(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create che subscription in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if created {
		return reconcile.Result{}, nil
	}

	if requeue, err := r.ensureWatchCheCluster(); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to add watch for CheCluster")
	} else if requeue {
		return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	}

	if created, statusMsg, err := r.ensureCheCluster(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create che cluster in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if created { // TODO VN: created can be removed
		return reconcile.Result{}, nil
	} else if statusMsg != "" {
		return reconcile.Result{}, r.statusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionInstalling, statusMsg)
	}

	return reconcile.Result{}, r.statusUpdate(reqLogger, cheInstallation, r.setStatusCheSubscriptionReady, "")
}

func (r *ReconcileCheInstallation) ensureCheNamespace(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheOpNamespace := cheInstallation.Spec.CheOperatorSpec.Namespace
	ns := &v1.Namespace{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheOpNamespace}, ns); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating a namespace for che operator", "Namespace", cheOpNamespace)
			namespace := NewNamespace(cheOpNamespace)
			if err := controllerutil.SetControllerReference(cheInstallation, namespace, r.scheme); err != nil {
				return false, err
			}
			if err := r.client.Create(context.TODO(), namespace); err != nil {
				if errors.IsAlreadyExists(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	if ns.Status.Phase != v1.NamespaceActive {
		logger.Info("namespace is not in active state", "namespace", ns.Name, "phase", ns.Status.Phase)
		return true, nil
	}
	return false, nil
}

func (r *ReconcileCheInstallation) ensureCheOperatorGroup(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheOg := &olmv1.OperatorGroup{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: OperatorGroupName, Namespace: cheInstallation.Spec.CheOperatorSpec.Namespace}, cheOg); err != nil {
		if errors.IsNotFound(err) {
			cheOg = NewOperatorGroup(cheInstallation.Spec.CheOperatorSpec.Namespace)
			logger.Info("Creating an operatorgroup for che", "OperatorGroup.Namespace", cheOg.Namespace, "OperatorGroup.Name", cheOg.Name)
			if err := controllerutil.SetControllerReference(cheInstallation, cheOg, r.scheme); err != nil {
				return false, err
			}
			if err := r.client.Create(context.TODO(), cheOg); err != nil {
				if errors.IsAlreadyExists(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (r *ReconcileCheInstallation) ensureCheSubscription(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	sub := &olmv1alpha1.Subscription{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: SubscriptionName, Namespace: cheInstallation.Spec.CheOperatorSpec.Namespace}, sub); err != nil {
		if errors.IsNotFound(err) {
			cheSub := NewSubscription(cheInstallation.Spec.CheOperatorSpec.Namespace)
			logger.Info("Creating a subscription for che", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name)
			if err := controllerutil.SetControllerReference(cheInstallation, cheSub, r.scheme); err != nil {
				return false, err
			}
			if err := r.client.Create(context.TODO(), cheSub); err != nil {
				if errors.IsAlreadyExists(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (r *ReconcileCheInstallation) ensureWatchCheCluster() (bool, error) {
	if r.watchCheCluster != nil {
		cheClusterCRD := &apiextnv1beta1.CustomResourceDefinition{}
		if err := r.client.Get(context.TODO(), types.NamespacedName{Name: "checlusters.org.eclipse.che"}, cheClusterCRD); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		if err := r.watchCheCluster(); err != nil {
			if meta.IsNoMatchError(err) {
				return true, nil
			}
			return false, err
		}
		r.watchCheCluster = nil // make sure watchCheCluster() should NOT be called afterwards
	}
	return false, nil
}

func (r *ReconcileCheInstallation) ensureCheCluster(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, string, error) {
	cluster := &orgv1.CheCluster{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: CheClusterName, Namespace: cheInstallation.Spec.CheOperatorSpec.Namespace}, cluster); err != nil {
		if errors.IsNotFound(err) {
			cluster = NewCheCluster(cheInstallation.Spec.CheOperatorSpec.Namespace)
			logger.Info("Creating CheCluster for che", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
			if err := controllerutil.SetControllerReference(cheInstallation, cluster, r.scheme); err != nil {
				return false, getCheClusterStatus(nil), err
			}
			if err := r.client.Create(context.TODO(), cluster); err != nil {
				if errors.IsAlreadyExists(err) {
					return false, getCheClusterStatus(nil), nil
				}
				return false, getCheClusterStatus(nil), err
			}
			return true, getCheClusterStatus(cluster), nil
		}
		return false, getCheClusterStatus(nil), err
	}
	return false, getCheClusterStatus(cluster), nil
}

func getCheClusterStatus(cluster *orgv1.CheCluster) string {
	if cluster == nil {
		return fmt.Sprintf("Status is unknown for CheCluster '%s'", CheClusterName)
	} else if cluster.Status == (orgv1.CheClusterStatus{}) {
		return fmt.Sprintf("Status is unknown for CheCluster '%s'", CheClusterName)
	} else if cluster.Status.CheClusterRunning != AvailableStatus {
		if !cluster.Status.DbProvisoned {
			return fmt.Sprintf("Provisioning Database for CheCluster '%s'", cluster.Name)
		} else if !cluster.Status.KeycloakProvisoned {
			return fmt.Sprintf("Provisioning Keycloak for CheCluster '%s'", cluster.Name)
		} else if !cluster.Status.OpenShiftoAuthProvisioned {
			return fmt.Sprintf("Provisioning OpenShiftoAuth for CheCluster '%s'", cluster.Name)
		} else if cluster.Status.DevfileRegistryURL == "" {
			return fmt.Sprintf("Provisioning DevfileRegistry for CheCluster '%s'", cluster.Name)
		} else if cluster.Status.PluginRegistryURL == "" {
			return fmt.Sprintf("Provisioning PluginRegistry for CheCluster '%s'", cluster.Name)
		} else if cluster.Status.CheURL == "" {
			return fmt.Sprintf("Provisioning CheServer for CheCluster '%s'", cluster.Name)
		} else {
			return fmt.Sprintf("CheCluster running status is '%s' for CheCluster '%s'", cluster.Status.CheClusterRunning, cluster.Name)
		}
	}
	return ""
}

// wrapErrorWithStatusUpdate wraps the error and update the install config status. If the update failed then logs the error.
func (r *ReconcileCheInstallation) wrapErrorWithStatusUpdate(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation, statusUpdater func(cheInstallation *v1alpha1.CheInstallation, message string) error, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	if err := statusUpdater(cheInstallation, err.Error()); err != nil {
		logger.Error(err, "status update failed")
	}
	return errs.Wrapf(err, format, args...)
}

func (r *ReconcileCheInstallation) statusUpdate(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation, statusUpdater func(cheInstallation *v1alpha1.CheInstallation, message string) error, msg string) error {
	if err := statusUpdater(cheInstallation, msg); err != nil {
		logger.Error(err, "unable to update status")
		return errs.Wrapf(err, "failed to update status")
	}
	return nil
}

func (r *ReconcileCheInstallation) updateStatusConditions(cheInstallation *v1alpha1.CheInstallation, newConditions ...toolchainv1alpha1.Condition) error {
	var updated bool
	cheInstallation.Status.Conditions, updated = condition.AddOrUpdateStatusConditions(cheInstallation.Status.Conditions, newConditions...)
	if !updated {
		// Nothing changed
		return nil
	}
	return r.client.Status().Update(context.TODO(), cheInstallation)
}

func (r *ReconcileCheInstallation) setStatusCheSubscriptionInstalling(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, SubscriptionInstalling(message))
}

func (r *ReconcileCheInstallation) setStatusCheSubscriptionFailed(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, SubscriptionFailed(message))
}

func (r *ReconcileCheInstallation) setStatusCheSubscriptionReady(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, SubscriptionCreated())
}
