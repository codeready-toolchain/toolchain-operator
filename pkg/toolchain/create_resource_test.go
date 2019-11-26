package toolchain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/tekton"
	"github.com/codeready-toolchain/toolchain-operator/pkg/test"
	"github.com/codeready-toolchain/toolchain-operator/pkg/toolchain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateFromYAML(t *testing.T) {

	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)

	t.Run("tekton", func(t *testing.T) {

		t.Run("create", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)
			ti, err := tekton.Asset("toolchain.openshift.dev_v1alpha1_tektoninstallation_cr.yaml")
			require.NoError(t, err)
			// when
			err = toolchain.CreateFromYAML(s, cl, ti)
			// then
			require.NoError(t, err)
			result := v1alpha1.TektonInstallation{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "tekton-installation"}, &result)
			assert.NoError(t, err)
		})

		t.Run("ignore if already exists", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)
			ti, err := tekton.Asset("toolchain.openshift.dev_v1alpha1_tektoninstallation_cr.yaml")
			require.NoError(t, err)
			err = toolchain.CreateFromYAML(s, cl, ti)
			require.NoError(t, err)
			// when (create again)
			err = toolchain.CreateFromYAML(s, cl, ti)
			// then
			require.NoError(t, err)
		})

		t.Run("fail for other reasons", func(t *testing.T) {
			// given
			cl := test.NewFakeClient(t)
			cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
				return fmt.Errorf("failed to create the obj")
			}
			ti, err := tekton.Asset("toolchain.openshift.dev_v1alpha1_tektoninstallation_cr.yaml")
			require.NoError(t, err)
			// when
			err = toolchain.CreateFromYAML(s, cl, ti)
			// then
			require.Error(t, err)
			assert.Equal(t, "failed to create the obj", err.Error())
		})

	})

}
