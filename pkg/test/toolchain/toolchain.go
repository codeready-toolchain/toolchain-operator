package toolchain

import (
	"fmt"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const TektonInstallation = "tekton-installation"

func NewCheInstallation(cheNamespace string) *v1alpha1.CheInstallation {
	cheInstallation := GenerateName("che-installation")
	return &v1alpha1.CheInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name: cheInstallation,
		},
		Spec: v1alpha1.CheInstallationSpec{
			CheOperatorSpec: v1alpha1.CheOperator{Namespace: cheNamespace},
		},
	}
}

func GenerateName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func NewTektonInstallation() *v1alpha1.TektonInstallation {
	return &v1alpha1.TektonInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name: TektonInstallation,
		},
	}
}
