package cheinstallation

import (
	"context"
	"fmt"
	"sync"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	commoncontroller "github.com/codeready-toolchain/toolchain-common/pkg/controller"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"

	che "github.com/eclipse/che-operator/pkg/apis/org/v1"
	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	errs "github.com/pkg/errors"
	"github.com/redhat-cop/operator-utils/pkg/util"
	v1 "k8s.io/api/core/v1"
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
		// make sure that there's a label with this key on the CheCluster in order to trigger a new reconcile loop
		return c.Watch(&source.Kind{Type: &orgv1.CheCluster{}}, commoncontroller.MapToOwnerByLabel("", "provider"))
	}

	err = watchCheCluster()
	if err != nil {
		if !meta.IsNoMatchError(err) { // ignore NoKindMatchError
			return err
		}
		log.Info("Postponing watcher on CheCluster resources")
		r.watchCheCluster = watchCheCluster
	} else {
		log.Info("Added a watcher on the CheCluster resources")
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
	mu              sync.Mutex
}

// Reconcile reads that state of the cluster for a CheInstallation object and makes changes based on the state read
// and what is in the CheInstallation.Spec
func (r *ReconcileCheInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CheInstallation")

	cheInstallation := &v1alpha1.CheInstallation{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name: InstallationName,
	}, cheInstallation)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("CheInstallation not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// ensure there's a finalizer, unless it's being deleted
	if !util.IsBeingDeleted(cheInstallation) {
		// Add the finalizer if it is not present
		if err := r.addFinalizer(reqLogger, cheInstallation); err != nil {
			return reconcile.Result{}, err
		}
	} else if util.HasFinalizer(cheInstallation, toolchainv1alpha1.FinalizerName) { // Che Installation is being deleted, but before that we should delete the Che Operator namespace explicitely
		reqLogger.Info("Terminating CheInstallation")
		if deleted, err := r.ensureCheClusterDeletion(reqLogger, cheInstallation); err != nil {
			return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationFailed, err, "failed to delete CheCluster resource in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
		} else if deleted {
			return reconcile.Result{}, r.setStatusCheInstallationTerminating(cheInstallation, "deleting CheCluster resource")
		} else {
			// CheCluster resource is already deleted, we can now remove the finalizer on the CheInstallation
			util.RemoveFinalizer(cheInstallation, toolchainv1alpha1.FinalizerName)
			if err := r.client.Update(context.Background(), cheInstallation); err != nil {
				return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationTerminating, err, "failed to remove finalizer")
			}
			return reconcile.Result{}, nil
		}
	}

	if requeue, err := r.ensureCheNamespace(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationFailed, err, "failed to create namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if requeue {
		return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	}

	if created, err := r.ensureCheOperatorGroup(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationFailed, err, "failed to create operatorgroup in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if created {
		return reconcile.Result{}, nil
	}

	if created, err := r.ensureCheSubscription(reqLogger, cheInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationFailed, err, "failed to create Che subscription in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	} else if created {
		return reconcile.Result{}, nil
	}

	if requeue, err := r.ensureWatchCheCluster(); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationFailed, err, "failed to add watch for CheCluster")
	} else if requeue {
		return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	}

	cheCluster, err := r.ensureCheCluster(reqLogger, cheInstallation)
	if err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationFailed, err, "failed to create Che cluster in namespace %s", cheInstallation.Spec.CheOperatorSpec.Namespace)
	}
	installed, msg := getCheClusterStatus(cheCluster)
	reqLogger.Info("checluster ensured", "msg", msg, "installed", installed)
	if !installed {
		return reconcile.Result{}, r.statusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationInstalling, msg)
	}

	reqLogger.Info("done with Che installation")
	return reconcile.Result{}, r.statusUpdate(reqLogger, cheInstallation, r.setStatusCheInstallationSucceeded(cheCluster), "")
}

// setFinalizers sets the finalizers for NSTemplateSet
func (r *ReconcileCheInstallation) addFinalizer(reqLogger logr.Logger, cheInstallation *v1alpha1.CheInstallation) error {
	// Add the finalizer if it is not present
	if !util.HasFinalizer(cheInstallation, toolchainv1alpha1.FinalizerName) {
		util.AddFinalizer(cheInstallation, toolchainv1alpha1.FinalizerName)
		reqLogger.Info("Adding finalizer on the CheInstallation resource")
		return r.client.Update(context.TODO(), cheInstallation)
	}
	return nil
}

func (r *ReconcileCheInstallation) ensureCheNamespace(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheOpNamespace := cheInstallation.Spec.CheOperatorSpec.Namespace
	namespace := NewNamespace(cheOpNamespace)
	if err := controllerutil.SetControllerReference(cheInstallation, namespace, r.scheme); err != nil {
		return false, err
	}
	if err := r.client.Create(context.TODO(), namespace); err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("Namespace for Che operator already exists", "Namespace", cheOpNamespace)
			ns := v1.Namespace{}
			if err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheOpNamespace}, &ns); err != nil {
				return false, err
			}
			if ns.Status.Phase != v1.NamespaceActive {
				logger.Info("Namespace is not in active state - deleting remaining CheCluster resource", "namespace", ns.Name, "phase", ns.Status.Phase)
				// return 'true' in any case, as we don't want to continue with the current reconciliation loop
				_, err = r.ensureCheClusterDeletion(logger, cheInstallation)
				return true, err
			}
			return false, nil
		}
		logger.Info("Unexpected error while creating a namespace for Che operator", "Namespace", cheOpNamespace, "message", err.Error())
		return false, err
	}
	logger.Info("Created a namespace for Che operator", "Namespace", cheOpNamespace)
	return true, nil

}

func (r *ReconcileCheInstallation) ensureCheOperatorGroup(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheOg := NewOperatorGroup(cheInstallation.Spec.CheOperatorSpec.Namespace)
	if err := controllerutil.SetControllerReference(cheInstallation, cheOg, r.scheme); err != nil {
		return false, err
	}
	if err := r.client.Create(context.TODO(), cheOg); err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("OperatorGroup for Che already exists", "OperatorGroup.Namespace", cheOg.Namespace, "OperatorGroup.Name", cheOg.Name)
			return false, nil
		}
		return false, err
	}
	logger.Info("Created an OperatorGroup for Che", "OperatorGroup.Namespace", cheOg.Namespace, "OperatorGroup.Name", cheOg.Name)
	return true, nil
}

func (r *ReconcileCheInstallation) ensureCheSubscription(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheSub := NewSubscription(cheInstallation.Spec.CheOperatorSpec.Namespace)
	if err := controllerutil.SetControllerReference(cheInstallation, cheSub, r.scheme); err != nil {
		return false, err
	}
	if err := r.client.Create(context.TODO(), cheSub); err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("Subscription for Che already exists", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name)
			return false, nil
		}
		logger.Info("Unexpected error while creating a Subscription for Che", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name, "message", err.Error())
		return false, err
	}
	logger.Info("Created a Subscription for Che", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name)
	return true, nil
}

// ensureWatchCheCluster adds watch for CheCluster resource if CheCluster CRD is installed else return requeue with true
// CheCluster CRD may takes time to get installed until CheOperator is installed successfully
// Once watch addded for CheCluster, sub-sequent calls to ensureWatchCheCluster() will do nothing
func (r *ReconcileCheInstallation) ensureWatchCheCluster() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.watchCheCluster != nil {
		if err := r.watchCheCluster(); err != nil {
			if meta.IsNoMatchError(err) {
				log.Info("CheGroup resource type does not exist yet", "message", err.Error())
				return true, nil
			}
			log.Info("Unexpected error while creating a watcher on the CheGroup resources", "message", err.Error())
			return false, err
		}
		log.Info("Added a watcher on the CheCluster resources")
		r.watchCheCluster = nil // make sure watchCheCluster() should NOT be called afterwards
	}
	log.Info("Watcher on the CheGroup resources already added")
	return false, nil
}

func (r *ReconcileCheInstallation) ensureCheCluster(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (*che.CheCluster, error) {
	cluster := NewCheCluster(cheInstallation.Spec.CheOperatorSpec.Namespace)
	if err := r.client.Create(context.TODO(), cluster); err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("CheCluster already exists", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
			cluster = &che.CheCluster{}
			if err = r.client.Get(context.TODO(), types.NamespacedName{Name: CheClusterName, Namespace: cheInstallation.Spec.CheOperatorSpec.Namespace}, cluster); err != nil {
				return nil, err
			}
			return cluster, nil
		}
		logger.Info("Unexpected error while creating a CheCluster for Che", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
		return nil, err
	}
	logger.Info("Created a CheCluster for Che", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
	return cluster, nil
}

func (r *ReconcileCheInstallation) ensureCheClusterDeletion(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cluster := &orgv1.CheCluster{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{
		Namespace: cheInstallation.Spec.CheOperatorSpec.Namespace,
		Name:      CheClusterName,
	}, cluster); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("CheCluster already deleted", "CheCluster.Namespace", cheInstallation.Spec.CheOperatorSpec.Namespace, "CheCluster.Name", CheClusterName)
			return false, nil
		}
		logger.Info("Unexpected error while creating a CheCluster for Che", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
		return false, err
	}
	logger.Info("Deleted CheCluster for Che", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
	return true, r.client.Delete(context.TODO(), cluster)
}

// getCheClusterStatus returns `true, ""` if the CheCluster is `cheClusterRunning: Available`,
// otherwise, it returns `false, <reason>`
func getCheClusterStatus(cluster *che.CheCluster) (bool, string) {
	if cluster == nil || cluster.Status == (che.CheClusterStatus{}) {
		return false, fmt.Sprintf("Status is unknown for CheCluster '%s'", CheClusterName)
	}
	if cluster.Status.CheClusterRunning == AvailableStatus {
		return true, ""
	}
	switch {
	case !cluster.Status.DbProvisoned:
		return false, fmt.Sprintf("Provisioning Database for CheCluster '%s'", cluster.Name)
	case !cluster.Status.KeycloakProvisoned:
		return false, fmt.Sprintf("Provisioning Keycloak for CheCluster '%s'", cluster.Name)
	case !cluster.Status.OpenShiftoAuthProvisioned:
		return false, fmt.Sprintf("Provisioning OpenShiftoAuth for CheCluster '%s'", cluster.Name)
	case cluster.Status.DevfileRegistryURL == "":
		return false, fmt.Sprintf("Provisioning DevfileRegistry for CheCluster '%s'", cluster.Name)
	case cluster.Status.PluginRegistryURL == "":
		return false, fmt.Sprintf("Provisioning PluginRegistry for CheCluster '%s'", cluster.Name)
	case cluster.Status.CheURL == "":
		return false, fmt.Sprintf("Provisioning CheServer for CheCluster '%s'", cluster.Name)
	default:
		return false, fmt.Sprintf("CheCluster running status is '%s' for CheCluster '%s'", cluster.Status.CheClusterRunning, cluster.Name)
	}
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

type updateStatusFunc func(cheInstallation *v1alpha1.CheInstallation, message string) error

func (r *ReconcileCheInstallation) statusUpdate(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation, updateStatus updateStatusFunc, msg string) error {
	if err := updateStatus(cheInstallation, msg); err != nil {
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

func (r *ReconcileCheInstallation) setStatusCheInstallationInstalling(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, Installing(message))
}

func (r *ReconcileCheInstallation) setStatusCheInstallationFailed(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, InstallationFailed(message))
}

func (r *ReconcileCheInstallation) setStatusCheInstallationTerminating(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, Terminating(message))
}

func (r *ReconcileCheInstallation) setStatusCheInstallationSucceeded(cheCluster *che.CheCluster) updateStatusFunc {
	return func(cheInstallation *v1alpha1.CheInstallation, message string) error {
		return r.updateStatusConditions(cheInstallation, InstallationSucceeded())
	}
}
