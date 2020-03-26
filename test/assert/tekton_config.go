package assert

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	config "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TektonConfigAssertion struct {
	t              *testing.T
	client         client.Reader
	namespacedName types.NamespacedName
	tektonConfig   *config.Config
}

func AssertThatTektonConfig(t *testing.T, name string, client client.Reader) *TektonConfigAssertion {
	return &TektonConfigAssertion{
		t:              t,
		client:         client,
		namespacedName: types.NamespacedName{Name: name},
	}
}

func (a *TektonConfigAssertion) Exists() *TektonConfigAssertion {
	err := a.loadTektonConfigAssertion()
	require.NoError(a.t, err)
	return a
}

func (a *TektonConfigAssertion) loadTektonConfigAssertion() error {
	config := &config.Config{}
	err := a.client.Get(context.TODO(), a.namespacedName, config)
	a.tektonConfig = config
	return err
}
