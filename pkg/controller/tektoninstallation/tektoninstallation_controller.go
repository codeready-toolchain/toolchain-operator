package tektoninstallation

import (
	"context"

	toolchainapiv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	toolchainv1alpha1 "github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/tekton"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"github.com/go-logr/logr"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	errs "github.com/pkg/errors"
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
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTektonInstallation{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

	return c.Watch(&source.Kind{Type: &olmv1alpha1.Subscription{}}, enqueueRequestForOwner)
}

// blank assignment to verify that ReconcileTektonInstallation implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTektonInstallation{}

// ReconcileTektonInstallation reconciles a TektonInstallation object
type ReconcileTektonInstallation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a TektonInstallation object and makes changes based on the state read
// and what is in the TektonInstallation.Spec
func (r *ReconcileTektonInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TektonInstallation")

	// Fetch the TektonInstallation instance
	tektonInstallation := &toolchainv1alpha1.TektonInstallation{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: toolchain.TektonInstallation}, tektonInstallation); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err := r.EnsureTektonSubscription(reqLogger, tektonInstallation)
	return reconcile.Result{}, err
}

func (r *ReconcileTektonInstallation) EnsureTektonSubscription(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation) error {
	tektonSubNamespace := tekton.SubscriptionNamespace
	if err := r.ensureTektonSubscription(logger, tektonInstallation, tektonSubNamespace); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, tektonInstallation, r.setStatusTektonSubscriptionFailed, err, "failed to create tekton subscription in namespace %s", tektonSubNamespace)
	}
	return r.StatusUpdate(logger, tektonInstallation, r.setStatusTektonSubscriptionReady, "")
}

func (r *ReconcileTektonInstallation) ensureTektonSubscription(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation, ns string) error {
	sub := &olmv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: tekton.SubscriptionName}, sub)
	if err != nil && errors.IsNotFound(err) {
		tektonSub := tekton.NewSubscription(ns)
		logger.Info("Creating subscription for tekton", "Subscription.Namespace", ns, "Subscription.Name", tektonSub.Name)
		if err := controllerutil.SetControllerReference(tektonInstallation, tektonSub, r.scheme); err != nil {
			return err
		}
		return r.client.Create(context.TODO(), tektonSub)
	}
	return err
}

func (r *ReconcileTektonInstallation) setStatusTektonSubscriptionReady(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, tekton.SubscriptionCreated())
}

func (r *ReconcileTektonInstallation) setStatusTektonSubscriptionFailed(tektonInstallation *v1alpha1.TektonInstallation, message string) error {
	return r.updateStatusConditions(tektonInstallation, tekton.SubscriptionFailed(message))
}

func (r *ReconcileTektonInstallation) StatusUpdate(logger logr.Logger, tektonInstallation *v1alpha1.TektonInstallation, statusUpdater func(tektonInstallation *v1alpha1.TektonInstallation, message string) error, msg string) error {
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
