module github.com/codeready-toolchain/toolchain-operator

require (
	github.com/codeready-toolchain/api v0.0.0-20191206004733-862cefb68396
	github.com/codeready-toolchain/toolchain-common v0.0.0-20191010043304-822e291d04cb
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.4
	github.com/gobuffalo/flect v0.1.7 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible // indirect
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190718033018-874b1785500e
	github.com/operator-framework/operator-sdk v0.11.1-0.20191012024916-f419ad3f3dc5
	github.com/pkg/errors v0.8.1
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.0.0-20191207000613-e7e4b65ae663 // indirect
	golang.org/x/sys v0.0.0-20191206220618-eeba5f6aabab // indirect
	golang.org/x/tools v0.0.0-20191206204035-259af5ff87bd // indirect
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55 // indirect
	google.golang.org/grpc v1.23.0 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20191120175047-4206685974f2 // indirect
	k8s.io/api v0.0.0
	k8s.io/apiextensions-apiserver v0.0.0 // indirect
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.0.0-20191121015212-c4c8f8345c7e
	k8s.io/gengo v0.0.0-20191120174120-e74f70b9b27e // indirect
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f // indirect
	sigs.k8s.io/controller-runtime v0.2.2
	sigs.k8s.io/controller-tools v0.2.4
)

// Pinned to kubernetes-1.14.1
replace (
	github.com/openshift/api => github.com/openshift/api v3.9.1-0.20190717200738-0390d1e77d64+incompatible
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20190627172412-c44a8b61b9f4
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20190525122359-d20e84d0fb64

	k8s.io/api => k8s.io/api v0.0.0-20190704095032-f4ca3d3bdf1d
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190704104557-6209bbe9f7a9
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190704094733-8f6ac2502e51
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190704101451-e5f5c6e528cd
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190409023024-d644b00f3b79
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190521191137-11646d1007e0+incompatible
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190409023720-1bc0c81fa51d
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190704094409-6c2a4329ac29
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190704100636-f0322db00a10
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190704101955-e796fd6d55e0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.5-beta.0.0.20190708100021-7936da50c68f
	sigs.k8s.io/controller-runtime v0.2.2 => sigs.k8s.io/controller-runtime v0.2.0
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.1
	sigs.k8s.io/structured-merge-diff => sigs.k8s.io/structured-merge-diff v0.0.0-20190302045857-e85c7b244fd2
)

replace github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.31.1

go 1.13
