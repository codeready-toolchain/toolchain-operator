package installconfig

import (
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-operator/pkg/che"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/k8s"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/olm"
	. "github.com/codeready-toolchain/toolchain-operator/pkg/test/toolchain"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestInstallConfigController(t *testing.T) {
	t.Run("reconcile with installconfig", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheOg := che.NewOperatorGroup(cheOperatorNs)
		cheSub := che.NewSubscription(cheOperatorNs)
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)

		request := newReconcileRequest(installConfig)

		// when
		_, err := r.Reconcile(request)

		// then
		assert.NoError(t, err)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(che.Labels())

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)

		AssertThatInstallConfig(t, installConfig.Namespace, installConfig.Name, cl).
			HasConditions(CheSubscriptionCreated("che operator subscription created"))
	})

	t.Run("do not reconcile without installconfig", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		cheOg := che.NewOperatorGroup(cheOperatorNs)
		cheSub := che.NewSubscription(cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs)

		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		request := newReconcileRequest(installConfig)

		// when
		_, err := r.Reconcile(request)

		// then
		assert.NoError(t, err)
		AssertThatNamespace(t, cheOperatorNs, cl).
			DoesNotExist()

		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			DoesNotExist()

		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			DoesNotExist()
	})

}

func TestCreateOperatorGroupForChe(t *testing.T) {
	t.Run("create operator group", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// when
		err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)

		//then
		assert.NoError(t, err)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)
	})

	t.Run("do not fail if operator group already exists", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheOg := che.NewOperatorGroup(cheOperatorNs)

		// create for the first time
		err := r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)
		assert.NoError(t, err)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)

		// when
		err = r.ensureCheOperatorGroup(testLogger(), cheOperatorNs, installConfig)

		// then
		assert.NoError(t, err)
		AssertThatOperatorGroup(t, cheOg.Namespace, cheOg.Name, cl).
			Exists().
			HasSize(1).
			HasSpec(cheOg.Spec)
	})

}

func TestCreateSubscriptionForChe(t *testing.T) {
	t.Run("create subscription", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheSub := che.NewSubscription(cheOperatorNs)

		// when
		err := r.createCheSubscription(testLogger(), cheOperatorNs, installConfig)

		// then
		assert.NoError(t, err)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)
	})

	t.Run("do not fail if subscription already exists", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)
		cheSub := che.NewSubscription(cheOperatorNs)

		// create for the first time
		err := r.createCheSubscription(testLogger(), cheOperatorNs, installConfig)
		assert.NoError(t, err)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)

		// when
		err = r.createCheSubscription(testLogger(), cheOperatorNs, installConfig)

		// then
		assert.NoError(t, err)
		AssertThatSubscription(t, cheSub.Namespace, cheSub.Name, cl).
			Exists().
			HasSpec(cheSub.Spec)
	})

}

func TestCreateNamespaceForChe(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		// given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)

		// when
		_, err := r.ensureCheNamespace(testLogger(), installConfig)

		// then
		assert.NoError(t, err)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(che.Labels())
	})

	t.Run("do not fail if ns exists", func(t *testing.T) {
		//given
		cheOperatorNs := GenerateName("che-op")
		installConfig := NewInstallConfig(GenerateName("toolchain-op"), cheOperatorNs)
		cl, r := configureClient(t, cheOperatorNs, installConfig)

		// create for the first time
		_, err := r.ensureCheNamespace(testLogger(), installConfig)
		assert.NoError(t, err)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(che.Labels())

		// when
		_, err = r.ensureCheNamespace(testLogger(), installConfig)

		// then
		assert.NoError(t, err)
		AssertThatNamespace(t, cheOperatorNs, cl).
			Exists().
			HasLabels(che.Labels())
	})

}

func configureClient(t *testing.T, cheOperatorNs string, initObjs ...runtime.Object) (client.Client, *ReconcileInstallConfig) {
	s := apiScheme(t)
	cl := fake.NewFakeClientWithScheme(s, initObjs...)
	reconcileInstallConfig := &ReconcileInstallConfig{scheme: s, client: cl}
	return cl, reconcileInstallConfig
}

func newReconcileRequest(installConfig *v1alpha1.InstallConfig) reconcile.Request {
	namespacedName := types.NamespacedName{Namespace: installConfig.Namespace, Name: installConfig.Name}
	return reconcile.Request{namespacedName}
}

func apiScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}

func testLogger() logr.Logger {
	logger := zap.Logger(true)
	logf.SetLogger(logger)
	return logger
}
