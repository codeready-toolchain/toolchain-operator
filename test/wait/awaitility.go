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
	Client    client.Client
}

// WaitForInstallConfig waits until there is InstallConfig with the given name available
func (a *ToolchainAwaitility) WaitForInstallConfig(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.InstallConfig{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of InstallConfig '%s'", name)
				return false, nil
			}
			return false, err
		}
		a.T.Logf("found InstallConfig '%s'", name)
		return true, nil
	})
}

func (a *ToolchainAwaitility) WaitForInstallConfigToDelete(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.InstallConfig{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("InstallConfig '%s' deleted", name)
				return true, nil
			}
			return false, err
		}
		a.T.Logf("waiting for deletion of InstallConfig '%s'", name)

		return false, nil
	})
}

func (a *ToolchainAwaitility) GetInstallConfig(name string) (*v1alpha1.InstallConfig, error) {
	ic := &v1alpha1.InstallConfig{}
	err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic)
	return ic, err
}

// InstallConfigWaitCondition represents a function checking if InstallConfig meets the given condition
type InstallConfigWaitCondition func(a *ToolchainAwaitility, ic *v1alpha1.InstallConfig) bool

// UntilHasStatusCondition checks if InstallConfig status has the given set of conditions
func UntilHasStatusCondition(conditions ...toolchainv1alpha1.Condition) InstallConfigWaitCondition {
	return func(a *ToolchainAwaitility, ic *v1alpha1.InstallConfig) bool {
		toolchain.AssertConditionsMatch(a.T, ic.Status.Conditions, conditions...)
		if toolchain.ConditionsMatch(ic.Status.Conditions, conditions...) {
			a.T.Logf("status conditions match in InstallConfig '%s`", ic.Name)
			return true
		}
		a.T.Logf("waiting for correct status condition of InstallConfig '%s`", ic.Name)
		return false
	}
}

// WaitForICConditions waits until there is InstallConfig available with the given name and meeting the set of given wait-conditions
func (a *ToolchainAwaitility) WaitForICConditions(name string, waitCond ...InstallConfigWaitCondition) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.InstallConfig{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: a.Namespace, Name: name}, ic); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of InstallConfig '%s'", name)
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
