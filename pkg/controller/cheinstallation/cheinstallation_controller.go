package cheinstallation

import (
	"context"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
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
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCheInstallation{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

	return nil
}

// blank assignment to verify that ReconcileCheInstallation implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCheInstallation{}

// ReconcileCheInstallation reconciles a CheInstallation object
type ReconcileCheInstallation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
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
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err = r.EnsureCheInstallation(reqLogger, cheInstallation)
	return reconcile.Result{}, err
}

func (r *ReconcileCheInstallation) EnsureCheInstallation(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) error {
	ns := cheInstallation.Spec.CheOperatorSpec.Namespace

	if created, err := r.ensureCheNamespace(logger, cheInstallation); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create namespace %s", ns)
	} else if created {
		return nil
	}

	if created, err := r.ensureCheOperatorGroup(logger, ns, cheInstallation); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create operatorgroup in namespace %s", ns)
	} else if created {
		return nil
	}

	if created, err := r.ensureCheSubscription(logger, ns, cheInstallation); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create che subscription in namespace %s", ns)
	} else if created {
		return nil
	}

	if created, err := r.ensureCheCluster(logger, ns, cheInstallation); err != nil {
		return r.wrapErrorWithStatusUpdate(logger, cheInstallation, r.setStatusCheSubscriptionFailed, err, "failed to create che cluster in namespace %s", ns)
	} else if created {
		return nil
	}

	return r.statusUpdate(logger, cheInstallation, r.setStatusCheSubscriptionReady, "")
}

func (r *ReconcileCheInstallation) ensureCheNamespace(logger logr.Logger, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheOpNamespace := cheInstallation.Spec.CheOperatorSpec.Namespace
	ns := &v1.Namespace{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: cheOpNamespace}, ns); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating a namespace for che operator", "Namespace", cheOpNamespace)
			namespace := che.NewNamespace(cheOpNamespace)
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

	// To handle if namespace is deleted by user and it's in terminating state and not in active state
	if ns.Status.Phase != v1.NamespaceActive {
		return false, errs.Errorf("namespace %s is not in active state", ns.Name)
	}
	return false, nil
}

func (r *ReconcileCheInstallation) ensureCheOperatorGroup(logger logr.Logger, ns string, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cheOg := &olmv1.OperatorGroup{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: ns, Namespace: ns}, cheOg); err != nil {
		if errors.IsNotFound(err) {
			cheOg = che.NewOperatorGroup(ns)
			logger.Info("Creating a operatorgroup for che", "OperatorGroup.Namespace", cheOg.Namespace, "OperatorGroup.Name", cheOg.Name)

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

func (r *ReconcileCheInstallation) ensureCheSubscription(logger logr.Logger, ns string, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	sub := &olmv1alpha1.Subscription{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: che.SubscriptionName, Namespace: ns}, sub); err != nil {
		if errors.IsNotFound(err) {
			cheSub := che.NewSubscription(ns)
			logger.Info("Creating subscription for che", "Subscription.Namespace", cheSub.Namespace, "Subscription.Name", cheSub.Name)
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

func (r *ReconcileCheInstallation) ensureCheCluster(logger logr.Logger, ns string, cheInstallation *v1alpha1.CheInstallation) (bool, error) {
	cluster := &orgv1.CheCluster{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: che.CheClusterName, Namespace: ns}, cluster); err != nil {
		if errors.IsNotFound(err) {
			cluster = che.NewCheCluster(ns)
			logger.Info("Creating CheCluster for che", "CheCluster.Namespace", cluster.Namespace, "CheCluster.Name", cluster.Name)
			if err := controllerutil.SetControllerReference(cheInstallation, cluster, r.scheme); err != nil {
				return false, err
			}
			if err := r.client.Create(context.TODO(), cluster); err != nil {
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

func (r *ReconcileCheInstallation) setStatusCheSubscriptionFailed(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, che.SubscriptionFailed(message))
}

func (r *ReconcileCheInstallation) setStatusCheSubscriptionReady(cheInstallation *v1alpha1.CheInstallation, message string) error {
	return r.updateStatusConditions(cheInstallation, che.SubscriptionCreated())
}
