package pkg

import (
	"context"

	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultOpenShiftOperatorsNamespace = "openshift-operators"
	codereadyToolchainPackageName      = "codeready-toolchain-operator"
)

// CreateInstallationResources creates both CheInstallation and TektonInstallation resources. If they already exist then it ignores it.
// Before the actual creation it also tries to get the Subscription that was created for codeready-toolchain-operator.
// If such a subscription is found, then it sets it as ownerReference for the Installation resources. The reason is
// that we need to remove(uninstall) the Che and Tekton operators when the codeready-toolchain-operator is being uninstalled,
// which means their respective Subscriptions are also removed. Thanks to the garbage collector it will ensure that both
// Che and Tekton operators will be uninstalled too.
func CreateInstallationResources(cl client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	tektonInstallation := tektoninstallation.NewInstallation()
	cheInstallation := cheinstallation.NewInstallation()

	subscriptions := &v1alpha1.SubscriptionList{}
	err := cl.List(context.TODO(), subscriptions, client.InNamespace(defaultOpenShiftOperatorsNamespace))
	if err != nil {
		return errors.Wrap(err, "unable to list the subscription")
	}
	for _, subscription := range subscriptions.Items {
		if subscription.Spec.Package == codereadyToolchainPackageName {
			if err := controllerutil.SetControllerReference(&subscription, tektonInstallation, scheme); err != nil {
				return errors.Wrap(err, "unable to set the owner reference to Tekton Installation")
			}
			if err := controllerutil.SetControllerReference(&subscription, cheInstallation, scheme); err != nil {
				return errors.Wrap(err, "unable to set the owner reference to Che Installation")
			}
			break
		}
	}

	// create the TektonInstallation resource on the cluster at startup, stop if something when wrong
	log.Info("Creating the Tekton installation resource")
	if err = cl.Create(context.TODO(), tektonInstallation); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "Failed to create the 'TektonInstallation' custom resource during startup")
	}
	log.Info("Tekton Installation resource created")

	// create the CheInstallation resource on the cluster at startup, stop if something when wrong
	log.Info("Creating the Che installation resource")
	if err = cl.Create(context.TODO(), cheInstallation); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "Failed to create the 'CheInstallation' custom resource during startup")
	}
	log.Info("Che Installation resource created")

	return nil
}
