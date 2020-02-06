package pkg

import (
	applyCl "github.com/codeready-toolchain/toolchain-common/pkg/client"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	applyClient := applyCl.NewApplyClient(cl, scheme)

	// we cannot set the owner reference for the *Installation resources because fo this issue: https://issues.redhat.com/browse/CRT-454

	// create the TektonInstallation resource, stop if something wrong happened
	log.Info("Creating the Tekton installation resource")
	if _, err := applyClient.CreateOrUpdateObject(tektonInstallation, true, nil); err != nil {
		return errors.Wrap(err, "Failed to create the 'TektonInstallation' custom resource")
	}
	log.Info("Tekton Installation resource created")

	// create the CheInstallation resource, stop if something wrong happened
	log.Info("Creating the Che installation resource")
	if _, err := applyClient.CreateOrUpdateObject(cheInstallation, true, nil); err != nil {
		return errors.Wrap(err, "Failed to create the 'CheInstallation' custom resource")
	}
	log.Info("Che Installation resource created")

	return nil
}
