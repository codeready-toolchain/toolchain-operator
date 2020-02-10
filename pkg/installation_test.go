package pkg_test

import (
	"testing"

	"github.com/codeready-toolchain/toolchain-operator/pkg"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/cheinstallation"
	"github.com/codeready-toolchain/toolchain-operator/pkg/controller/tektoninstallation"
	"github.com/codeready-toolchain/toolchain-operator/test"
	testolm "github.com/codeready-toolchain/toolchain-operator/test/olm"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	t.Run("when subscription is available then assign", func(t *testing.T) {
		// given
		client := test.NewFakeClient(t, subscription)

		// when
		err = pkg.CreateInstallationResources(client, s, logf.Log)

		// then
		require.NoError(t, err)
		testolm.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasOwnerRef(subscription)
		testolm.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasOwnerRef(subscription)
	})

	t.Run("when subscription is not available then don't assign", func(t *testing.T) {
		// given
		client := test.NewFakeClient(t)

		// when
		err = pkg.CreateInstallationResources(client, s, logf.Log)

		// then
		require.NoError(t, err)
		testolm.AssertThatTektonInstallation(t, "", tektoninstallation.InstallationName, client).
			HasNoOwnerRef()
		testolm.AssertThatCheInstallation(t, "", cheinstallation.InstallationName, client).
			HasNoOwnerRef()
	})
}
