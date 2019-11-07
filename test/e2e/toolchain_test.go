package e2e

import (
	"context"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	"github.com/codeready-toolchain/toolchain-operator/pkg/tekton"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/k8s"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/olm"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	"github.com/codeready-toolchain/toolchain-operator/test/wait"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestToolchain(t *testing.T) {
	testType := os.Getenv(test.TestType)
	defer func() {
		err := os.Setenv(test.TestType, testType)
		require.NoError(t, err, "failed to restore env variable %s=%s", test.TestType, testType)
	}()
	err := os.Setenv(test.TestType, test.E2e)
	require.NoError(t, err, "failed to set env variable %s=%s", test.TestType, test.E2e)

	ctx, await := InitOperator(t)
	defer ctx.Cleanup()
	cheOperatorNs := GenerateName("che-op")
	cheOg := che.NewOperatorGroup(cheOperatorNs)
	cheSub := che.NewSubscription(cheOperatorNs)
	tektonSub := tekton.NewSubscription(tekton.SubscriptionNamespace)

	cheInstallation := NewCheInstallation(cheOperatorNs)
	tektonInstallation := NewTektonInstallation()

	f := framework.Global

	t.Run("should create operator group and subscription for che with CheInstallation", func(t *testing.T) {
		// when
		err := f.Client.Create(context.TODO(), cheInstallation, cleanupOptions(ctx))

		// then
		require.NoError(t, err, "failed to create toolchain CheInstallation")

		err = await.WaitForCheInstallConditions(cheInstallation.Name, wait.UntilHasCheStatusCondition(che.SubscriptionCreated(che.SubscriptionSuccess)))
		require.NoError(t, err)

		AssertThatNamespace(t, cheOperatorNs, f.Client).
			Exists().
			HasLabels(toolchain.Labels())

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			Exists().
			HasSpec(cheSub.Spec)
	})

	t.Run("should recreate che operator's ns operatorgroup subscription when ns deleted", func(t *testing.T) {
		// given
		ns := &v1.Namespace{}
		err := f.Client.Get(context.TODO(), types.NamespacedName{Name: cheOperatorNs}, ns)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), ns)

		// then
		require.NoError(t, err, "failed to delete Che Operator Namespace")

		err = await.WaitForNamespace(cheOperatorNs)
		require.NoError(t, err)

		err = await.WaitForCheInstallConditions(cheInstallation.Name, wait.UntilHasCheStatusCondition(che.SubscriptionCreated(che.SubscriptionSuccess)))
		require.NoError(t, err)

		AssertThatNamespace(t, cheOperatorNs, f.Client).
			Exists().
			HasLabels(toolchain.Labels())

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			Exists().
			HasSpec(cheSub.Spec)
	})

	t.Run("should recreate deleted operatorgroup for che", func(t *testing.T) {
		// given
		ogList := &olmv1.OperatorGroupList{}
		err := await.Client.List(context.TODO(), ogList, client.InNamespace(cheOperatorNs), client.MatchingLabels(toolchain.Labels()))
		require.NoError(t, err)
		require.Len(t, ogList.Items, 1)

		// when
		err = f.Client.Delete(context.TODO(), ogList.Items[0].DeepCopy())

		// then
		require.NoError(t, err, "failed to delete OperatorGroup %s from namespace %s", ogList.Items[0].Name, cheOperatorNs)

		err = await.WaitForOperatorGroup(cheOperatorNs, toolchain.Labels())
		require.NoError(t, err)

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)
	})

	t.Run("should recreate deleted subscription for che", func(t *testing.T) {
		// given
		cheSubscription, err := await.GetSubscription(cheSub.Namespace, cheSub.Name)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), cheSubscription)

		// then
		require.NoError(t, err, "failed to delete CheSubscription %s in namespace %s", cheSubscription.Name, cheSubscription.Namespace)

		err = await.WaitForSubscription(cheSub.Namespace, cheSub.Name)
		require.NoError(t, err)

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			Exists().
			HasSpec(cheSub.Spec)
	})

	t.Run("should remove operatorgroup and subscription for che with CheInstallation deletion", func(t *testing.T) {
		// given
		cheInstallation, err := await.GetCheInstallation(cheInstallation.Name)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), cheInstallation)

		// then
		require.NoError(t, err, "failed to create toolchain CheInstallation")

		err = await.WaitForCheInstallationToDelete(cheInstallation.Name)
		require.NoError(t, err)

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			DoesNotExist()

		AssertThatNamespace(t, cheOperatorNs, f.Client).
			DoesNotExist()
	})

	t.Run("should create subscription for tekton with TektonInstallation", func(t *testing.T) {
		// when
		err := f.Client.Create(context.TODO(), tektonInstallation, cleanupOptions(ctx))

		// then
		require.NoError(t, err, "failed to create toolchain TektonInstallation")

		err = await.WaitForTektonInstallConditions(tektonInstallation.Name, wait.UntilHasTektonStatusCondition(tekton.SubscriptionCreated(tekton.SubscriptionSuccess)))
		require.NoError(t, err)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, f.Client).
			Exists().
			HasSpec(tektonSub.Spec)
	})

	t.Run("should recreate deleted subscription for tekton", func(t *testing.T) {
		// given
		tektonSubscription, err := await.GetSubscription(tekton.SubscriptionNamespace, tekton.SubscriptionName)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), tektonSubscription)

		// then
		require.NoError(t, err, "failed to delete TektonInstallation")

		err = await.WaitForSubscription(tekton.SubscriptionNamespace, tekton.SubscriptionName)
		require.NoError(t, err)

		err = await.WaitForTektonInstallConditions(tektonInstallation.Name, wait.UntilHasTektonStatusCondition(tekton.SubscriptionCreated(tekton.SubscriptionSuccess)))
		require.NoError(t, err)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, f.Client).
			Exists().
			HasSpec(tektonSub.Spec)
	})
}

func InitOperator(t *testing.T) (*framework.TestCtx, wait.ToolchainAwaitility) {
	icList := &v1alpha1.CheInstallationList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, icList)
	require.NoError(t, err, "failed to add custom resource scheme to framework: %v", err)

	t.Parallel()
	ctx := framework.NewTestCtx(t)

	err = ctx.InitializeClusterResources(cleanupOptions(ctx))
	require.NoError(t, err, "failed to initialize cluster resources")

	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "failed to get namespace where operator is running")

	// get global framework variables
	f := framework.Global
	await := wait.ToolchainAwaitility{
		T:         t,
		Namespace: namespace,
		Client:    f.Client,
	}
	// wait for toolchain-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "toolchain-operator", 1, wait.RetryInterval, wait.Timeout)
	require.NoError(t, err, "failed while waiting for toolchain-operator deployment")
	t.Log("toolchain-operator is ready and running state")

	return ctx, await
}

func cleanupOptions(ctx *framework.TestCtx) *framework.CleanupOptions {
	return &framework.CleanupOptions{TestContext: ctx, Timeout: wait.CleanupTimeout, RetryInterval: wait.CleanupRetryInterval}
}
