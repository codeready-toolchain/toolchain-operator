package apis

import (
	"github.com/codeready-toolchain/api/pkg/apis"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	// add olm Subscription Schema
	addToSchemes := append(apis.AddToSchemes, olmv1alpha1.AddToScheme, olmv1.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme)

	return addToSchemes.AddToScheme(s)
}
