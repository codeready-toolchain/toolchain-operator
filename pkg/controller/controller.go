package controller

import (
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, cheinstallation.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, tektoninstallation.Add)
}

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
