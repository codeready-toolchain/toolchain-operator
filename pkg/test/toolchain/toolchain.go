package toolchain

import (
	"fmt"
	"github.com/codeready-toolchain/toolchain-operator/pkg/apis/toolchain/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func NewInstallConfig(toolchainOp string, cheNamespace string) *v1alpha1.InstallConfig {
	installConfig := GenerateName("install-cfg")
	return &v1alpha1.InstallConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installConfig,
			Namespace: toolchainOp,
		},
		Spec: v1alpha1.InstallConfigSpec{
			CheOperatorSpec: v1alpha1.CheOperator{Namespace: cheNamespace},
		},
	}
}

func GenerateName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
