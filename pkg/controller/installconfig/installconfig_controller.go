package installconfig

import (
	"context"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	errs "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

const (
	// Status condition reasons
	FailedToCreateCheSubscriptionReason = "FailedToCreateCheSubscription"
	CreatedCheSubscriptionReason        = "CreatedCheSubscription"
)

var log = logf.Log.WithName("controller_installconfig")

// Add creates a new InstallConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInstallConfig{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("installconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource InstallConfig
	err = c.Watch(&source.Kind{Type: &v1alpha1.InstallConfig{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileInstallConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileInstallConfig{}

// ReconcileInstallConfig reconciles a InstallConfig object
type ReconcileInstallConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a InstallConfig object and makes changes based on the state read
// and what is in the InstallConfig.Spec
// Note: The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileInstallConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling InstallConfig")

	// Fetch the InstallConfig instance
	installConfig := &v1alpha1.InstallConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, installConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := r.EnsureCheSubscription(reqLogger, installConfig); err != nil {
		return reconcile.Result{}, err
	}

	return r.StatusUpdate(reqLogger, installConfig, r.setStatusReady, "che operator subscription created")
}

// wrapErrorWithStatusUpdate wraps the error and update the install config status. If the update failed then logs the error.
func (r *ReconcileInstallConfig) wrapErrorWithStatusUpdate(logger logr.Logger, installConfig *v1alpha1.InstallConfig, statusUpdater func(installConfig *v1alpha1.InstallConfig, message string) error, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	if err := statusUpdater(installConfig, err.Error()); err != nil {
		logger.Error(err, "status update failed")
	}
	return errs.Wrapf(err, format, args...)
}

func (r *ReconcileInstallConfig) EnsureCheSubscription(logger logr.Logger, installConfig *v1alpha1.InstallConfig) error {
	ns, err := r.ensureCheNamespace(logger, installConfig)
	if err != nil {
		return r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusCheSubscriptionFailed, err, "failed to create namespace %s", installConfig.Spec.CheOperatorSpec.Namespace)
	}

	if err := r.ensureCheOperatorGroup(logger, ns, installConfig); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusCheSubscriptionFailed, err, "failed to create operatorgroup in namespace %s", ns)
	}

	if err := r.createCheSubscription(logger, ns, installConfig); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusCheSubscriptionFailed, err, "failed to create che subscription in namespace %s", ns)
	}
	return nil
}

func (r *ReconcileInstallConfig) ensureCheNamespace(logger logr.Logger, installConfig *v1alpha1.InstallConfig) (string, error) {
	cheOpNamespace := installConfig.Spec.CheOperatorSpec.Namespace
	ns := &v1.Namespace{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheOpNamespace}, ns)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a namespace for che operator", "Namespace", cheOpNamespace)
		namespace := che.NewNamespace(cheOpNamespace)
		if err := controllerutil.SetControllerReference(installConfig, namespace, r.scheme); err != nil {
			return cheOpNamespace, err
		}
		return cheOpNamespace, r.client.Create(context.TODO(), namespace)
	}

	return cheOpNamespace, err
}

func (r *ReconcileInstallConfig) ensureCheOperatorGroup(logger logr.Logger, ns string, installConfig *v1alpha1.InstallConfig) error {
	operatorGroup := che.NewOperatorGroup(ns)
	if err := controllerutil.SetControllerReference(installConfig, operatorGroup, r.scheme); err != nil {
		return err
	}

	ogList := &olmv1.OperatorGroupList{}
	err := r.client.List(context.TODO(), ogList, client.InNamespace(ns), client.MatchingLabels(che.Labels()))
	if err == nil && len(ogList.Items) == 0 {
		logger.Info("Creating a operatorgroup for che", "OperatorGroup.Namespace", operatorGroup.Namespace)
		return r.client.Create(context.TODO(), operatorGroup)
	}
	return err
}

func (r *ReconcileInstallConfig) createCheSubscription(logger logr.Logger, ns string, installConfig *v1alpha1.InstallConfig) error {
	cheSub := che.NewSubscription(ns)
	if err := controllerutil.SetControllerReference(installConfig, cheSub, r.scheme); err != nil {
		return err
	}
	sub := &olmv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheSub.GetName(), Namespace: cheSub.GetNamespace()}, sub)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating subscription for che", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name)
		return r.client.Create(context.TODO(), cheSub)
	}
	return nil
}

func (r *ReconcileInstallConfig) StatusUpdate(logger logr.Logger, installConfig *v1alpha1.InstallConfig, statusUpdater func(installConfig *v1alpha1.InstallConfig, message string) error, msg string) (reconcile.Result, error) {
	if err := statusUpdater(installConfig, msg); err != nil {
		logger.Error(err, "unable to update status")
		return reconcile.Result{
			RequeueAfter: time.Second,
			Requeue:      true,
		}, nil
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileInstallConfig) updateStatusConditions(installConfig *v1alpha1.InstallConfig, newConditions ...toolchainv1alpha1.Condition) error {
	var updated bool
	installConfig.Status.Conditions, updated = condition.AddOrUpdateStatusConditions(installConfig.Status.Conditions, newConditions...)
	if !updated {
		// Nothing changed
		return nil
	}
	return r.client.Status().Update(context.TODO(), installConfig)
}

func (r *ReconcileInstallConfig) setStatusCheSubscriptionFailed(installConfig *v1alpha1.InstallConfig, message string) error {
	return r.updateStatusConditions(
		installConfig,
		toolchainv1alpha1.Condition{
			Status:  v1.ConditionFalse,
			Reason:  FailedToCreateCheSubscriptionReason,
			Message: message,
		})
}

func (r *ReconcileInstallConfig) setStatusReady(installConfig *v1alpha1.InstallConfig, message string) error {
	return r.updateStatusConditions(
		installConfig,
		CheSubscriptionCreated(message))
}

func CheSubscriptionCreated(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    toolchainv1alpha1.ConditionReady,
		Status:  v1.ConditionTrue,
		Reason:  CreatedCheSubscriptionReason,
		Message: message,
	}
}
