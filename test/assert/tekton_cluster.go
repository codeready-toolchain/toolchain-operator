package assert

import (
	"context"
	"testing"

	config "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"

	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TektonClusterAssertion struct {
	t              *testing.T
	client         client.Reader
	namespacedName types.NamespacedName
	tektonCluster  *config.Config
}

func AssertThatTektonCluster(t *testing.T, name string, client client.Reader) *TektonClusterAssertion {
	return &TektonClusterAssertion{
		t:              t,
		client:         client,
		namespacedName: types.NamespacedName{Name: name},
	}
}

func (a *TektonClusterAssertion) Exists() *TektonClusterAssertion {
	err := a.loadTektonClusterAssertion()
	require.NoError(a.t, err)
	return a
}

func (a *TektonClusterAssertion) loadTektonClusterAssertion() error {
	cluster := &config.Config{}
	err := a.client.Get(context.TODO(), a.namespacedName, cluster)
	a.tektonCluster = cluster
	return err
}
