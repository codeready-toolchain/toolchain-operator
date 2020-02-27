package pkg_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-operator/pkg"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
	"github.com/codeready-toolchain/toolchain-operator/test"
	"github.com/codeready-toolchain/toolchain-operator/test/assert"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestCreateInstallationResources(t *testing.T) {
	// given
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)

	t.Run("when the CheInstallation or TektonInstallation resources are not present then it creates them", func(t *testing.T) {
		// given

		client := test.NewFakeClient(t)

		// when
		err = pkg.CreateInstallationResources(client, s, logf.Log)

		// then
		require.NoError(t, err)
		assert.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasNoOwnerRef()
		assert.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
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
		assert.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasNoOwnerRef()
		assert.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasNoOwnerRef()
	})
}
