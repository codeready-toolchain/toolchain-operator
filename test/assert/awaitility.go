package assert

import (
	"context"
	"os"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/test"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Timeout              = time.Minute * 60
	RetryInterval        = time.Second * 5
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 5
)

type ToolchainAwaitility struct {
	T      *testing.T
	Client client.Reader
}

// WaitForCheInstallation waits until there is CheInstallation with the given name available
func (a *ToolchainAwaitility) WaitForCheInstallation(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.CheInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, ic); err != nil {
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

func (a *ToolchainAwaitility) WaitForCheInstallationToBeDeleted(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ic := &v1alpha1.CheInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, ic); err != nil {
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

func (a *ToolchainAwaitility) WaitForTektonInstallationToBeDeleted(name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		tektonInstallation := &v1alpha1.TektonInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, tektonInstallation); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("TektonInstallation '%s' deleted", name)
				return true, nil
			}
			return false, err
		}
		a.T.Logf("waiting for deletion of TektonInstallation '%s'", name)

		return false, nil
	})
}

func (a *ToolchainAwaitility) GetCheInstallation(name string) (*v1alpha1.CheInstallation, error) {
	ic := &v1alpha1.CheInstallation{}
	err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, ic)
	return ic, err
}

func (a *ToolchainAwaitility) GetSubscription(ns, name string) (*olmv1alpha1.Subscription, error) {
	subscription := &olmv1alpha1.Subscription{}
	err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, subscription)
	return subscription, err
}

func (a *ToolchainAwaitility) GetCheCluster(ns, name string) (*orgv1.CheCluster, error) {
	cheCluster := &orgv1.CheCluster{}
	err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, cheCluster)
	return cheCluster, err
}

// CheInstallationWaitCondition represents a function checking if CheInstallation meets the given condition
type CheInstallationWaitCondition func(a *ToolchainAwaitility, ic *v1alpha1.CheInstallation) bool

// TektonInstallationWaitCondition represents a function checking if TektonInstallation meets the given condition
type TektonInstallationWaitCondition func(a *ToolchainAwaitility, ic *v1alpha1.TektonInstallation) bool

// UntilHasCheStatusCondition checks if CheInstallation status has the given set of conditions
func UntilHasCheStatusCondition(conditions ...toolchainv1alpha1.Condition) CheInstallationWaitCondition {
	return func(a *ToolchainAwaitility, ic *v1alpha1.CheInstallation) bool {
		if len(ic.Status.Conditions) > 0 {
			if ConditionsMatch(ic.Status.Conditions, conditions...) {
				a.T.Logf("status conditions match in CheInstallation '%s`", ic.Name)
				return true
			}
		}
		a.T.Logf("waiting for correct status condition of CheInstallation '%s`", ic.Name)
		return false
	}
}

// UntilHasTektonStatusCondition checks if TektonInstallation status has the given set of conditions
func UntilHasTektonStatusCondition(conditions ...toolchainv1alpha1.Condition) TektonInstallationWaitCondition {
	return func(a *ToolchainAwaitility, ic *v1alpha1.TektonInstallation) bool {
		if ConditionsMatch(ic.Status.Conditions, conditions...) {
			a.T.Logf("status conditions match in TektonInstallation '%s`", ic.Name)
			return true
		}
		a.T.Logf("waiting for correct status condition of TektonInstallation '%s`", ic.Name)
		return false
	}
}

// WaitForCheInstallConditions waits until there is CheInstallation available with the given name and meeting the set of given wait-conditions
func (a *ToolchainAwaitility) WaitForCheInstallConditions(name string, waitCond ...CheInstallationWaitCondition) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ci := &v1alpha1.CheInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, ci); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of CheInstallation '%s'", name)
				return false, nil
			}
			return false, err
		}
		for _, isMatched := range waitCond {
			if !isMatched(a, ci) {
				return false, nil
			}
		}
		return true, nil
	})
}

// WaitForTektonInstallConditions waits until there is TektonInstallation available with the given name and meeting the set of given wait-conditions
func (a *ToolchainAwaitility) WaitForTektonInstallConditions(name string, waitCond ...TektonInstallationWaitCondition) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ti := &v1alpha1.TektonInstallation{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, ti); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of TektonInstallation '%s'", name)
				return false, nil
			}
			return false, err
		}
		for _, isMatched := range waitCond {
			if !isMatched(a, ti) {
				return false, nil
			}
		}
		return true, nil
	})
}

// WaitForSubscription waits until there is Subscription available with the given name and namespace
func (a *ToolchainAwaitility) WaitForSubscription(ns, name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		sub := &olmv1alpha1.Subscription{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, sub); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of Subscription '%s' in namespace '%s'", name, ns)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// WaitForNamespace waits until there is Namespace available with the given name
func (a *ToolchainAwaitility) WaitForNamespace(name string, expectedPhase v1.NamespacePhase) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ns := &v1.Namespace{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Name: name}, ns); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of namespace '%s'", name)
				return false, nil
			}
			return false, err
		}
		return ns.Status.Phase == expectedPhase, nil
	})
}

// WaitForOperatorGroup waits until there is OperatorGroup available with the given name and namespace
func (a *ToolchainAwaitility) WaitForOperatorGroup(ns string, labels map[string]string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		ogList := &olmv1.OperatorGroupList{}
		if err = a.Client.List(context.TODO(), ogList, client.InNamespace(ns), client.MatchingLabels(labels)); err != nil {
			return false, err
		}
		if len(ogList.Items) > 0 {
			return true, nil
		}
		a.T.Logf("waiting for availability of OperatorGroup with labels '%v' in namespace '%s'", labels, ns)
		return false, nil
	})
}

// WaitForCheCluster waits until there is CheCluster available with the given name and namespace
func (a *ToolchainAwaitility) WaitForCheCluster(ns, name string) error {
	return wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		cluster := &orgv1.CheCluster{}
		if err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, cluster); err != nil {
			if errors.IsNotFound(err) {
				a.T.Logf("waiting for availability of CheCluster '%s' in namespace '%s'", name, ns)
				return false, nil
			}
			return false, err
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
