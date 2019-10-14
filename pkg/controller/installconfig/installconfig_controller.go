package installconfig

import (
	"context"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/utils/che"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
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
	FailedToCreateCheSubscription = "FailedToCreateCheSubscription"
	CreatedCheSubscription        = "CreatedCheSubscription"

	ConditionFailed  toolchainv1alpha1.ConditionType = "Failed"
	ConditionCreated toolchainv1alpha1.ConditionType = "Created"
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
		reqLogger.Error(err, "failed to create subscription for che", "Che.Namespace", installConfig.Spec.CheOperatorSpec.Namespace)
		return r.StatusUpdate(reqLogger, installConfig, r.setStatusCheSubscriptionFailed, err.Error())
	}

	return r.StatusUpdate(reqLogger, installConfig, r.setStatusCheSubscriptionCreated, "che operator subscription created")
}

func (r *ReconcileInstallConfig) EnsureCheSubscription(logger logr.Logger, installConfig *v1alpha1.InstallConfig) error {
	ns, err := r.ensureCheNamespace(logger, installConfig)
	if err != nil {
		return err
	}

	if err := r.createCheOperatorGroup(logger, ns, installConfig); err != nil {
		return err
	}

	return r.createCheSubscription(logger, ns, installConfig)
}

func (r *ReconcileInstallConfig) ensureCheNamespace(logger logr.Logger, installConfig *v1alpha1.InstallConfig) (string, error) {
	cheNamespace := installConfig.Spec.CheOperatorSpec.Namespace
	ns := &v1.Namespace{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheNamespace}, ns)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a namespace for che operator", "Namespace", cheNamespace)
		namespace := che.Namespace(cheNamespace)
		if err := controllerutil.SetControllerReference(installConfig, namespace, r.scheme); err != nil {
			return cheNamespace, err
		}
		return cheNamespace, r.client.Create(context.TODO(), namespace)
	}

	return cheNamespace, err
}

func (r *ReconcileInstallConfig) createCheOperatorGroup(logger logr.Logger, ns string, installConfig *v1alpha1.InstallConfig) error {
	operatorGroup := che.OperatorGroup(ns)
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
	cheSub := che.Subscription(ns)
	if err := controllerutil.SetControllerReference(installConfig, cheSub, r.scheme); err != nil {
		return err
	}
	found := &olmv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheSub.GetName(), Namespace: cheSub.GetNamespace()}, found)
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
			Type:    ConditionFailed,
			Status:  v1.ConditionFalse,
			Reason:  FailedToCreateCheSubscription,
			Message: message,
		})
}

func (r *ReconcileInstallConfig) setStatusCheSubscriptionCreated(installConfig *v1alpha1.InstallConfig, message string) error {
	return r.updateStatusConditions(
		installConfig,
		CheSubscriptionCreated(message))
}

func CheSubscriptionCreated(message string) toolchainv1alpha1.Condition {
	return toolchainv1alpha1.Condition{
		Type:    ConditionCreated,
		Status:  v1.ConditionTrue,
		Reason:  CreatedCheSubscription,
		Message: message,
	}
}
