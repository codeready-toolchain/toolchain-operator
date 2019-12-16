// +build tools

package tools

import (
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
