package wait

import (
	"context"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

const (
	RetryInterval        = time.Second * 5
	Timeout              = time.Second * 60
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 5
)

type ToolchainAwaitility struct {
	T         *testing.T
	Namespace string
	Client    client.Reader
}

// WaitForCheInstallation waits until there is CheInstallation with the given name available
func (a *ToolchainAwaitility) WaitForCheInstallation(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.CheInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of CheInstallation '%s'", name)
				return false, nil
			}
			return false, err
		}
		a.T.Logf("found CheInstallation '%s'", name)
		return true, nil
	})
}

func (a *ToolchainAwaitility) WaitForCheInstallationToDelete(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.CheInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("CheInstallation '%s' deleted", name)
				return true, nil
			}
			return false, err
		}
		a.T.Logf("waiting for deletion of CheInstallation '%s'", name)

		return false, nil
	})
}

func (a *ToolchainAwaitility) GetCheInstallation(name string) (*v1alpha1.CheInstallation, error) {
	ic := &v1alpha1.CheInstallation{}
	err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic)
	return ic, err
}

// CheInstallationWaitCondition represents a function checking if CheInstallation meets the given condition
type CheInstallationWaitCondition func(a *ToolchainAwaitility, ic *v1alpha1.CheInstallation) bool

// UntilHasStatusCondition checks if CheInstallation status has the given set of conditions
func UntilHasStatusCondition(conditions ...toolchainv1alpha1.Condition) CheInstallationWaitCondition {
	return func(a *ToolchainAwaitility, ic *v1alpha1.CheInstallation) bool {
		toolchain.AssertConditionsMatch(a.T, ic.Status.Conditions, conditions...)
		if toolchain.ConditionsMatch(ic.Status.Conditions, conditions...) {
			a.T.Logf("status conditions match in CheInstallation '%s`", ic.Name)
			return true
		}
		a.T.Logf("waiting for correct status condition of CheInstallation '%s`", ic.Name)
		return false
	}
}

// WaitForCheInstallConditions waits until there is CheInstallation available with the given name and meeting the set of given wait-conditions
func (a *ToolchainAwaitility) WaitForCheInstallConditions(name string, waitCond ...CheInstallationWaitCondition) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.CheInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of CheInstallation '%s'", name)
				return false, nil
			}
			return false, err
		}
		for _, isMatched := range waitCond {
			if !isMatched(a, ic) {
				return false, nil
			}
		}
		return true, nil
	})
}

func PollOnceOrUntilCondition(condition func() (done bool, err error)) error {
	tt := os.Getenv(test.TestType)
	if tt == test.E2e {
		return wait.Poll(RetryInterval, Timeout, condition)
	}
	_, err := condition()
	return err
}
