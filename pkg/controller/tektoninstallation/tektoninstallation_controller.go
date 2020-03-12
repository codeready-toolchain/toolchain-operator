package tektoninstallation

import (
	"context"
	"sync"
	"time"

	commoncontroller "github.com/codeready-toolchain/toolchain-common/pkg/controller"

	"k8s.io/apimachinery/pkg/api/meta"

	toolchainapiv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	toolchainv1alpha1 "github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"

	"github.com/go-logr/logr"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	errs "github.com/pkg/errors"

	config "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_tektoninstallation")

// Add creates a new TektonInstallation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileTektonInstallation {
	return &ReconcileTektonInstallation{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileTektonInstallation) error {
	// Create a new controller
	c, err := controller.New("tektoninstallation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TektonInstallation
	if err := c.Watch(&source.Kind{Type: &v1alpha1.TektonInstallation{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	// Watch for changes to secondary resource
	enqueueRequestForOwner := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &toolchainv1alpha1.TektonInstallation{},
	}

	watchTektonCluster := func() error {
		// make sure that there's a label with this key on the CheCluster in order to trigger a new reconcile loop
		return c.Watch(&source.Kind{Type: &config.Config{}}, commoncontroller.MapToOwnerByLabel("", "provider"))
	}

	cluster := &config.Config{}
	err = mgr.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "deafult"}, cluster)
	if err != nil {
		if meta.IsNoMatchError(err) { // TektonCluster CRD is not installed yet. Postpone the watcher.
			log.Info("Postponing watcher on TektonCluster resources")
			r.watchTektonCluster = watchTektonCluster
		} else if !errors.IsNotFound(err) { // ignore NotFound
			return err
		}
	} else {
		log.Info("Added a watcher on the TektonCluster resources")
	}

	return c.Watch(&source.Kind{Type: &olmv1alpha1.Subscription{}}, enqueueRequestForOwner)

}

// blank assignment to verify that ReconcileTektonInstallation implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTektonInstallation{}

// ReconcileTektonInstallation reconciles a TektonInstallation object
type ReconcileTektonInstallation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client             client.Client
	scheme             *runtime.Scheme
	watchTektonCluster func() error
	mu                 sync.Mutex
}

// Reconcile reads that state of the cluster for a TektonInstallation object and makes changes based on the state read
// and what is in the TektonInstallation.Spec
func (r *ReconcileTektonInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TektonInstallation")
	// Fetch the TektonInstallation instance
	tektonInstallation := &toolchainv1alpha1.TektonInstallation{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: InstallationName}, tektonInstallation); err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("TektonInstallation not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := r.EnsureTektonSubscription(reqLogger, tektonInstallation); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, tektonInstallation, r.setStatusTektonSubscriptionFailed, err, "failed to create Tekton subscription in namespace %s", tektonInstallation.Name)
	}

	if requeue, err := r.ensureWatchTektonCluster(); err != nil {
		return reconcile.Result{}, r.wrapErrorWithStatusUpdate(reqLogger, tektonInstallation, r.setStatusTektonInstallationFailed, err, "failed to add watch for TektonCluster")
	} else if requeue {
		return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	}

	cluster := &config.Config{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: TektonClusterName}, cluster)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}, nil
	}

	status := getTektonClusterStatus(cluster)
	switch status {
	case config.InstalledStatus:
		reqLogger.Info("done with Tekton installation")
		return reconcile.Result{}, r.statusUpdate(reqLogger, tektonInstallation, r.setStatusTektonInstallationSucceeded, "")
	case config.InstallingStatus:
		return reconcile.Result{}, r.statusUpdate(reqLogger, tektonInstallation, r.setStatusTektonInstallationInstalling, "")
	case config.ErrorStatus:
		return reconcile.Result{}, r.statusUpdate(reqLogger, tektonInstallation, r.setStatusTektonInstallationFailed, "")
	default:
		return reconcile.Result{}, r.statusUpdate(reqLogger, tektonInstallation, r.setStatusTektonInstallationUnknown, "")
	}
}

// EnsureTektonSubscription ensures that there is an OLM Subscription resource for Tekton
func (r *ReconcileTektonInstallation) EnsureTektonSubscription(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation) error {
	tektonSubNamespace := SubscriptionNamespace
	if err := r.ensureTektonSubscription(logger, tektonInstallation, tektonSubNamespace); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, tektonInstallation, r.setStatusTektonSubscriptionFailed, err, "failed to create tekton subscription in namespace %s", tektonSubNamespace)
	}
	return nil
}

func (r *ReconcileTektonInstallation) ensureTektonSubscription(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation, ns string) error {
	sub := &olmv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: SubscriptionName}, sub)
	if err != nil && errors.IsNotFound(err) {
		tektonSub := NewSubscription(ns)
		logger.Info("Creating subscription for tekton", "Subscription.Namespace", ns, "Subscription.Name", tektonSub.Name)
		if err := controllerutil.SetControllerReference(tektonInstallation, tektonSub, r.scheme); err != nil {
			return err
		}
		return r.client.Create(context.TODO(), tektonSub)
	}

	return err
}

func (r *ReconcileTektonInstallation) ensureWatchTektonCluster() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.watchTektonCluster != nil {
		configCondition := &config.Config{}
		if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "default"}, configCondition); err != nil {
			if meta.IsNoMatchError(err) {
				log.Info("Tekton resource type does not exist yet", "message", err.Error())
				return true, nil
			}
			if !errors.IsNotFound(err) { // ignore NotFound
				log.Info("Unexpected error while getting a TektonCluster to ensure a TektonCluster watcher can be created", "message", err.Error())
				return false, err
			}
		}
		if err := r.watchTektonCluster(); err != nil {
			log.Info("Unexpected error while creating a watcher on the Tekton resources", "message", err.Error())
			return false, err
		}
		log.Info("Added a watcher on the TektonCluster resources")
		r.watchTektonCluster = nil // make sure watchTektonCluster() should NOT be called afterwards
	}
	log.Info("Watcher on the Tekton resources already added")
	return false, nil
}

func getTektonClusterStatus(cluster *config.Config) config.InstallStatus {
	var status config.InstallStatus = "unknown"
	for _, conditions := range cluster.Status.Conditions {
		code := conditions.Code
		if code == config.InstalledStatus || code == config.InstallingStatus || code == config.ErrorStatus {
			status = code
			break
		}
	}
	return status
}

func (r *ReconcileTektonInstallation) setStatusTektonInstallationSucceeded(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, InstallationSucceeded())
}

func (r *ReconcileTektonInstallation) setStatusTektonInstallationInstalling(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, InstallationInstalling())
}

func (r *ReconcileTektonInstallation) setStatusTektonInstallationFailed(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, InstallationFailed(message))
}

func (r *ReconcileTektonInstallation) setStatusTektonInstallationUnknown(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, InstallationUnknown())
}

func (r *ReconcileTektonInstallation) setStatusTektonSubscriptionFailed(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, InstallationFailed(message))
}

func (r *ReconcileTektonInstallation) statusUpdate(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation, statusUpdater func(tektonInstallation *v1alpha1.TektonInstallation, message string) error, msg string) error {
	if err := statusUpdater(tektonInstallation, msg); err != nil {
		logger.Error(err, "unable to update status")
		return errs.Wrapf(err, "failed to update status")
	}
	return nil
}

func (r *ReconcileTektonInstallation) updateStatusConditions(tektonInstallation *v1alpha1.TektonInstallation, newConditions ...toolchainapiv1alpha1.Condition) error {
	var updated bool
	tektonInstallation.Status.Conditions, updated = condition.AddOrUpdateStatusConditions(tektonInstallation.Status.Conditions, newConditions...)
	if !updated {
		// Nothing changed
		return nil
	}
	return r.client.Status().Update(context.TODO(), tektonInstallation)
}

// wrapErrorWithStatusUpdate wraps the error and update the install config status. If the update failed then logs the error.
func (r *ReconcileTektonInstallation) wrapErrorWithStatusUpdate(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation, statusUpdater func(cheInstallation *v1alpha1.TektonInstallation, message string) error, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	if err := statusUpdater(tektonInstallation, err.Error()); err != nil {
		logger.Error(err, "status update failed")
	}
	return errs.Wrapf(err, format, args...)
}
