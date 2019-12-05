package apis

import (
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	apiextnv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

func init() {
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, olmv1alpha1.AddToScheme)
	AddToSchemes = append(AddToSchemes, olmv1.AddToScheme)
	AddToSchemes = append(AddToSchemes, orgv1.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, apiextnv1beta1.AddToScheme)
}

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
