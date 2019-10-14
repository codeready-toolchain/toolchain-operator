package e2e

import (
	"context"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/installconfig"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/k8s"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/olm"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"github.com/codeready-toolchain/toolchain-operator/pkg/utils/che"
	"github.com/codeready-toolchain/toolchain-operator/test/wait"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestToolchain(t *testing.T) {
	ctx, await := InitOperator(t)
	defer ctx.Cleanup()
	cheOperatorNs := toolchain.GenerateName("che-op")
	cheOg := che.OperatorGroup(cheOperatorNs)
	cheSub := che.Subscription(cheOperatorNs)
	installcfg := NewInstallConfig(await.Namespace, cheOperatorNs)
	f := framework.Global

	t.Run("should create operator group and subscription for che with installconfig", func(t *testing.T) {
		// when
		err := f.Client.Create(context.TODO(), installcfg, cleanupOptions(ctx))

		// then
		require.NoError(t, err, "failed to create toolchain InstallConfig")

		err = await.WaitForInstallConfig(installcfg.Name)
		require.NoError(t, err)

		err = await.WaitForICConditions(installcfg.Name, wait.UntilHasStatusCondition(installconfig.CheSubscriptionCreated("che operator subscription created")))
		require.NoError(t, err)

		AssertThatNamespace(t, cheOperatorNs, f.Client).
			Exists().
			HasLabels(che.Labels())

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			Exists().
			HasSpec(cheSub.Spec)
	})

	t.Run("should remove operatorgroup and subscription for che with installconfig deletion", func(t *testing.T) {
		// given
		installConfig, err := await.GetInstallConfig(installcfg.Name)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), installConfig)

		// then
		require.NoError(t, err, "failed to create toolchain InstallConfig")

		err = await.WaitForInstallConfigToDelete(installcfg.Name)
		require.NoError(t, err)

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			DoesNotExist()

		AssertThatNamespace(t, cheOperatorNs, f.Client).
			DoesNotExist()
	})
}

func InitOperator(t *testing.T) (*framework.TestCtx, wait.ToolchainAwaitility) {
	icList := &v1alpha1.InstallConfigList{}
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
