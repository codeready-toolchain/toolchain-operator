package controller

import (
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, tektoninstallation.Add)
}
