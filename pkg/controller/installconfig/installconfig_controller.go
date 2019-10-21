package installconfig

import (
	"context"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	"github.com/codeready-toolchain/toolchain-operator/pkg/tekton"
	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	errs "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
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

	installConfig := &v1alpha1.InstallConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, installConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var errors []error
	if isResourceCreated, err := r.EnsureCheSubscription(reqLogger, installConfig); err != nil {
		errors = append(errors, err)
	} else if isResourceCreated {
		return reconcile.Result{Requeue: true}, nil
	}

	if isResourceCreated, err := r.EnsureTektonSubscription(reqLogger, installConfig); err != nil {
		errors = append(errors, err)
	} else if isResourceCreated {
		return reconcile.Result{Requeue: true}, nil
	}

	if len(errors) > 0 {
		return reconcile.Result{}, errorsutil.NewAggregate(errors)
	}

	return reconcile.Result{}, nil
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

func (r *ReconcileInstallConfig) EnsureTektonSubscription(logger logr.Logger, installConfig *v1alpha1.InstallConfig) (bool, error) {
	tektonSubNamespace := tekton.SubscriptionNamespace
	if subCreated, err := r.ensureTektonSubscription(logger, tektonSubNamespace, installConfig); err != nil {
		return subCreated, r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusTektonSubscriptionFailed, err, "failed to create tekton subscription in namespace %s", tektonSubNamespace)
	} else if subCreated {
		return subCreated, nil
	}

	return false, r.StatusUpdate(logger, installConfig, r.setStatusTektonSubscriptionReady, tekton.SubscriptionSuccess)
}

func (r *ReconcileInstallConfig) EnsureCheSubscription(logger logr.Logger, installConfig *v1alpha1.InstallConfig) (bool, error) {
	ns := installConfig.Spec.CheOperatorSpec.Namespace
	if nsCreated, err := r.ensureCheNamespace(logger, installConfig); err != nil {
		return nsCreated, r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusCheSubscriptionFailed, err, "failed to create namespace %s", ns)
	} else if nsCreated {
		return nsCreated, nil
	}

	if ogCreated, err := r.ensureCheOperatorGroup(logger, ns, installConfig); err != nil {
		return ogCreated, r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusCheSubscriptionFailed, err, "failed to create operatorgroup in namespace %s", ns)
	} else if ogCreated {
		return ogCreated, nil
	}

	if subCreated, err := r.ensureCheSubscription(logger, ns, installConfig); err != nil {
		return subCreated, r.wrapErrorWithStatusUpdate(logger, installConfig, r.setStatusCheSubscriptionFailed, err, "failed to create che subscription in namespace %s", ns)
	} else if subCreated {
		return subCreated, nil
	}

	return false, r.StatusUpdate(logger, installConfig, r.setStatusCheSubscriptionReady, che.SubscriptionSuccess)
}

func (r *ReconcileInstallConfig) ensureCheNamespace(logger logr.Logger, installConfig *v1alpha1.InstallConfig) (bool, error) {
	cheOpNamespace := installConfig.Spec.CheOperatorSpec.Namespace
	nsCreated := false
	ns := &v1.Namespace{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheOpNamespace}, ns)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a namespace for che operator", "Namespace", cheOpNamespace)
		namespace := che.NewNamespace(cheOpNamespace)
		if err := controllerutil.SetControllerReference(installConfig, namespace, r.scheme); err != nil {
			return nsCreated, err
		}
		if err := r.client.Create(context.TODO(), namespace); err != nil {
			return nsCreated, err
		}
		return true, nil
	}

	return nsCreated, err
}

func (r *ReconcileInstallConfig) ensureCheOperatorGroup(logger logr.Logger, ns string, installConfig *v1alpha1.InstallConfig) (bool, error) {
	operatorGroup := che.NewOperatorGroup(ns)
	ogCreated := false
	if err := controllerutil.SetControllerReference(installConfig, operatorGroup, r.scheme); err != nil {
		return ogCreated, err
	}

	ogList := &olmv1.OperatorGroupList{}
	err := r.client.List(context.TODO(), ogList, client.InNamespace(ns), client.MatchingLabels(che.Labels()))
	if err == nil && len(ogList.Items) == 0 {
		logger.Info("Creating a operatorgroup for che", "OperatorGroup.Namespace", operatorGroup.Namespace)
		if err := r.client.Create(context.TODO(), operatorGroup); err != nil {
			return ogCreated, err
		}
		return true, nil
	}
	return ogCreated, err
}

func (r *ReconcileInstallConfig) ensureCheSubscription(logger logr.Logger, ns string, installConfig *v1alpha1.InstallConfig) (bool, error) {
	cheSub := che.NewSubscription(ns)
	subCreated := false
	if err := controllerutil.SetControllerReference(installConfig, cheSub, r.scheme); err != nil {
		return subCreated, err
	}
	sub := &olmv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheSub.GetName(), Namespace: cheSub.GetNamespace()}, sub)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating subscription for che", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name)
		if err := r.client.Create(context.TODO(), cheSub); err != nil {
			return subCreated, err
		}
		return true, nil
	}
	return subCreated, err
}

func (r *ReconcileInstallConfig) ensureTektonSubscription(logger logr.Logger, ns string, installConfig *v1alpha1.InstallConfig) (bool, error) {
	tektonSub := tekton.NewSubscription(ns)
	subCreated := false
	if err := controllerutil.SetControllerReference(installConfig, tektonSub, r.scheme); err != nil {
		return subCreated, err
	}
	sub := &olmv1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: tektonSub.GetName(), Namespace: tektonSub.GetNamespace()}, sub)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating subscription for tekton", "Subscription.Namespace", tektonSub.Namespace, "Subscription.Name", tektonSub.Name)
		if err := r.client.Create(context.TODO(), tektonSub); err != nil {
			return subCreated, err
		}
		return true, nil
	}
	return subCreated, err
}

func (r *ReconcileInstallConfig) StatusUpdate(logger logr.Logger, installConfig *v1alpha1.InstallConfig, statusUpdater func(installConfig *v1alpha1.InstallConfig, message string) error, msg string) error {
	if err := statusUpdater(installConfig, msg); err != nil {
		logger.Error(err, "unable to update status")
		return errs.Wrapf(err, "failed to update status")
	}
	return nil
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
	return r.updateStatusConditions(installConfig, che.SubscriptionFailed(message))
}

func (r *ReconcileInstallConfig) setStatusTektonSubscriptionFailed(installConfig *v1alpha1.InstallConfig, message string) error {
	return r.updateStatusConditions(installConfig, tekton.SubscriptionFailed(message))
}

func (r *ReconcileInstallConfig) setStatusCheSubscriptionReady(installConfig *v1alpha1.InstallConfig, message string) error {
	return r.updateStatusConditions(installConfig, che.SubscriptionCreated(message))
}

func (r *ReconcileInstallConfig) setStatusTektonSubscriptionReady(installConfig *v1alpha1.InstallConfig, message string) error {
	return r.updateStatusConditions(installConfig, tekton.SubscriptionCreated(message))
}
