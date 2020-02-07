package pkg_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-operator/pkg"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestCreateInstallationResources(t *testing.T) {
	// given
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)

	subscription := &v1alpha1.Subscription{
		TypeMeta: v1.TypeMeta{
			Kind:       "Subscription",
			APIVersion: "operators.coreos.com/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "openshift-operators",
			Name:      "codeready-toolchain-operator",
			UID:       types.UID(uuid.NewV4().String()),
		},
		Spec: &v1alpha1.SubscriptionSpec{
			Package: "codeready-toolchain-operator",
		},
	}

	t.Run("when the *Installation resources are not present then it creates them", func(t *testing.T) {
		// given

		client := test.NewFakeClient(t)

		// when
		err = pkg.CreateInstallationResources(client, s, logf.Log)

		// then
		require.NoError(t, err)
		toolchain.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasNoOwnerRef()
		toolchain.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasNoOwnerRef()
	})

	t.Run("when subscription is set as owner reference then it should be removed", func(t *testing.T) {
		// given
		tektonInstallation := tektoninstallation.NewInstallation()
		err := controllerutil.SetControllerReference(subscription, tektonInstallation, s)
		require.NoError(t, err)
		cheInstallation := cheinstallation.NewInstallation()
		err = controllerutil.SetControllerReference(subscription, cheInstallation, s)
		require.NoError(t, err)

		client := test.NewFakeClient(t, subscription, tektonInstallation, cheInstallation)

		// when
		err = pkg.CreateInstallationResources(client, s, logf.Log)

		// then
		require.NoError(t, err)
		toolchain.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasNoOwnerRef()
		toolchain.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasNoOwnerRef()
	})

	t.Run("when owner reference is not set then it doesn't add anything", func(t *testing.T) {
		// given
		tektonInstallation := tektoninstallation.NewInstallation()
		cheInstallation := cheinstallation.NewInstallation()

		client := test.NewFakeClient(t, tektonInstallation, cheInstallation)

		// when
		err = pkg.CreateInstallationResources(client, s, logf.Log)

		// then
		require.NoError(t, err)
		toolchain.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasNoOwnerRef()
		toolchain.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasNoOwnerRef()
	})
}
