package e2e

import (
	"context"
	"os"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"
	"github.com/codeready-toolchain/toolchain-operator/test"
	. "github.com/codeready-toolchain/toolchain-operator/test/assert"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/stretchr/testify/require"
	config "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestToolchain(t *testing.T) {
	testType := os.Getenv(test.TestType)
	defer func() {
		err := os.Setenv(test.TestType, testType)
		require.NoError(t, err, "failed to restore env variable %s=%s", test.TestType, testType)
	}()
	err := os.Setenv(test.TestType, test.E2e)
	require.NoError(t, err, "failed to set env variable %s=%s", test.TestType, test.E2e)

	// ctx, await := InitOperator(t)
	_, await := InitOperator(t)
	// defer ctx.Cleanup()
	cheInstallation := cheinstallation.NewInstallation()
	cheOperatorNS := cheInstallation.Spec.CheOperatorSpec.Namespace
	cheOg := cheinstallation.NewOperatorGroup(cheOperatorNS)
	cheSub := cheinstallation.NewSubscription(cheOperatorNS)
	cheCluster := cheinstallation.NewCheCluster(cheOperatorNS)
	tektonInstallation := tektoninstallation.NewInstallation()
	installedCode := []config.ConfigCondition{
		config.ConfigCondition{
			Code: config.InstalledStatus,
		},
	}
	tektoninstallation.NewTektonCluster(installedCode...)
	tektonSub := tektoninstallation.NewSubscription(tektoninstallation.SubscriptionNamespace)

	f := framework.Global

	t.Run("should create operator group and subscription for Che with CheInstallation", func(t *testing.T) {
		// when
		// CheInstallation should already exist

		// then
		require.NoError(t, err, "failed to create toolchain CheInstallation")

		err = await.WaitForCheInstallConditions(cheInstallation.Name, UntilHasCheStatusCondition(cheinstallation.InstallationSucceeded()))
		require.NoError(t, err)
		checkCheResources(t, f.Client.Client, cheOperatorNS, cheOg, cheSub, cheCluster)
	})

	// TODO enable this test. The issue is Namespace when deleted, stuck in Terminating Phase
	// t.Run("should recreate Che operator ns operatorgroup subscription when ns deleted", func(t *testing.T) {
	// 	// given
	// 	ns := &v1.Namespace{}
	// 	err := f.Client.Get(context.TODO(), types.NamespacedName{Name: cheOperatorNS}, ns)
	// 	require.NoError(t, err)

	// 	// when
	// 	err = f.Client.Delete(context.TODO(), ns)

	// 	// then
	// 	require.NoError(t, err, "failed to delete Che Operator Namespace")

	// 	err = await.WaitForNamespace(cheOperatorNS, v1.NamespaceActive)
	// 	require.NoError(t, err)

	// 	err = await.WaitForCheInstallConditions(cheInstallation.Name, UntilHasCheStatusCondition(cheinstallation.InstallationSucceeded()))
	// 	require.NoError(t, err)
	// 	checkCheResources(t, f.Client.Client, cheOperatorNS, cheOg, cheSub, cheCluster)
	// })

	t.Run("should recreate deleted operator group for che", func(t *testing.T) {
		// given
		ogList := &olmv1.OperatorGroupList{}
		err := await.Client.List(context.TODO(), ogList, client.InNamespace(cheOperatorNS), client.MatchingLabels(toolchain.Labels()))
		require.NoError(t, err)
		require.Len(t, ogList.Items, 1)

		// when
		err = f.Client.Delete(context.TODO(), ogList.Items[0].DeepCopy())

		// then
		require.NoError(t, err, "failed to delete OperatorGroup %s from namespace %s", ogList.Items[0].Name, cheOperatorNS)

		err = await.WaitForOperatorGroup(cheOperatorNS, toolchain.Labels())
		require.NoError(t, err)
		checkCheResources(t, f.Client.Client, cheOperatorNS, cheOg, nil, nil)
	})

	t.Run("should recreate deleted subscription for Che", func(t *testing.T) {
		// given
		cheSubscription, err := await.GetSubscription(cheSub.Namespace, cheSub.Name)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), cheSubscription)

		// then
		require.NoError(t, err, "failed to delete CheSubscription %s in namespace %s", cheSubscription.Name, cheSubscription.Namespace)

		err = await.WaitForSubscription(cheSub.Namespace, cheSub.Name)
		require.NoError(t, err)
		checkCheResources(t, f.Client.Client, cheOperatorNS, nil, cheSub, nil)
	})

	t.Run("should recreate deleted checluster for Che", func(t *testing.T) {
		// given
		cluster, err := await.GetCheCluster(cheCluster.Namespace, cheCluster.Name)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), cluster)

		// then
		require.NoError(t, err, "failed to delete CheCluster %s in namespace %s", cluster.Name, cluster.Namespace)

		err = await.WaitForCheInstallConditions(cheInstallation.Name, UntilHasCheStatusCondition(cheinstallation.InstallationSucceeded()))
		require.NoError(t, err)
		checkCheResources(t, f.Client.Client, cheOperatorNS, cheOg, cheSub, cheCluster)
	})

	t.Run("should create subscription for tekton with TektonInstallation", func(t *testing.T) {
		// given
		// TektonInstallation should already exist
		// when
		err = await.WaitForTektonInstallConditions(tektonInstallation.Name, UntilHasTektonStatusCondition(tektoninstallation.InstallationSucceeded()))
		// then
		require.NoError(t, err)

		checkTektonResources(t, f.Client.Client, tektonSub)
	})

	t.Run("should recreate deleted subscription for tekton", func(t *testing.T) {
		// given
		tektonSubscription, err := await.GetSubscription(tektoninstallation.SubscriptionNamespace, tektoninstallation.SubscriptionName)
		require.NoError(t, err)

		// when
		err = f.Client.Delete(context.TODO(), tektonSubscription)

		// then
		require.NoError(t, err, "failed to delete TektonInstallation")

		err = await.WaitForSubscription(tektoninstallation.SubscriptionNamespace, tektoninstallation.SubscriptionName)
		require.NoError(t, err)

		err = await.WaitForTektonInstallConditions(tektonInstallation.Name, UntilHasTektonStatusCondition(tektoninstallation.InstallationSucceeded()))
		require.NoError(t, err)

		checkTektonResources(t, f.Client.Client, tektonSub)
	})

	t.Run("should remove both Tekton and Che when CheInstallation and TektonInstallation are deleted", func(t *testing.T) {
		// when
		err = f.Client.Delete(context.TODO(), cheInstallation)
		err = f.Client.Delete(context.TODO(), tektonInstallation)

		// then
		require.NoError(t, err)

		err = await.WaitForTektonInstallationToBeDeleted(tektonInstallation.Name)
		require.NoError(t, err)

		AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, f.Client).
			DoesNotExist()

		err = await.WaitForCheInstallationToBeDeleted(cheInstallation.Name)
		require.NoError(t, err)

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, f.Client).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, f.Client).
			DoesNotExist()

		AssertThatNamespace(t, cheOperatorNS, f.Client).
			DoesNotExist()
	})
}

func checkCheResources(t *testing.T, client client.Client, cheOperatorNs string, cheOg *olmv1.OperatorGroup, cheSub *olmv1alpha1.Subscription, cheCluster *orgv1.CheCluster) {
	t.Helper()
	AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
		HasFinalizer(toolchainv1alpha1.FinalizerName)
	AssertThatNamespace(t, cheOperatorNs, client).
		Exists().
		HasLabels(toolchain.Labels())

	if cheOg != nil {
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, client).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)
	}
	if cheSub != nil {
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, client).
			Exists().
			HasSpec(cheSub.Spec)
	}
	if cheCluster != nil {
		cheCluster := AssertThatCheCluster(t, cheCluster.Namespace, cheCluster.Name, client).
			Exists().
			HasRunningStatus(cheinstallation.AvailableStatus).
			Get()
		AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasServerURL(cheCluster.Status.CheURL)
	}
}

func checkTektonResources(t *testing.T, client client.Client, tektonSub *olmv1alpha1.Subscription) {
	t.Helper()
	AssertThatSubscription(t, tektonSub.Namespace, tektonSub.Name, client).
		Exists().
		HasSpec(tektonSub.Spec)
	AssertThatTektonCluster(t, tektoninstallation.TektonClusterName, client)
}

func InitOperator(t *testing.T) (*framework.TestCtx, ToolchainAwaitility) {
	icList := &v1alpha1.CheInstallationList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, icList)
	require.NoError(t, err, "failed to add custom resource scheme to framework: %v", err)

	ctx := framework.NewTestCtx(t)

	err = ctx.InitializeClusterResources(cleanupOptions(ctx))
	require.NoError(t, err, "failed to initialize cluster resources")

	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "failed to get namespace where operator is running")

	// get global framework variables
	f := framework.Global
	await := ToolchainAwaitility{
		T:      t,
		Client: f.Client,
	}
	// wait for toolchain-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "toolchain-operator", 1, RetryInterval, OperatorTimeout)
	require.NoError(t, err, "failed while waiting for toolchain-operator deployment")
	t.Log("toolchain-operator is ready and running state")

	return ctx, await
}

func cleanupOptions(ctx *framework.TestCtx) *framework.CleanupOptions {
	return &framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval}
}
