CSV_VERSION_TO_GENERATE := 0.1.4
DATE_SUFFIX := $(shell date +'%d%H%M%S')
COMMUNITY_OPERATORS_DIR=../../operator-framework/community-operators/community-operators/codeready-toolchain-operator

PATH_TO_CREATE_RELEASE_FILE= scripts/create-release-bundle.sh
PATH_TO_PUSH_MANIFEST_FILE= scripts/push-to-quay-manifest.sh
PATH_TO_CREATE_HACK_FILE= scripts/generate-deploy-hack.sh
PATH_TO_PUSH_NIGHTLY_FILE=scripts/push-to-quay-nightly.sh

PHONY: create-release-manifest
## Creates release manifest in ./manifest/ directory
create-release-manifest:
	$(eval CREATE_PARAMS = -pr ../toolchain-operator -on codeready-toolchain-operator  --next-version ${CSV_VERSION_TO_GENERATE} --quay-namespace codeready-toolchain -ch alpha)
ifneq ("$(wildcard ../api/$(PATH_TO_CREATE_RELEASE_FILE))","")
	@echo "creating release manifest in ./manifest/ directory using script from local api repo..."
	../api/${PATH_TO_CREATE_RELEASE_FILE} ${CREATE_PARAMS}
else
	@echo "creating release manifest in ./manifest/ directory using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_CREATE_RELEASE_FILE} | bash -s -- ${CREATE_PARAMS}
endif

PHONY: push-latest-release-manifest
## Pushes the latest release manifest from ./manifest/ directory into quay namespace
push-latest-release-manifest:
	$(eval PUSH_PARAMS = -pr ../toolchain-operator -on codeready-toolchain-operator)
ifneq ("$(wildcard ../api/$(PATH_TO_PUSH_MANIFEST_FILE))","")
	@echo "pushing the latest release manifest from ./manifest/ directory using script from local api repo..."
	../api/${PATH_TO_PUSH_MANIFEST_FILE} ${PUSH_PARAMS}
else
	@echo "ushing the latest release manifest from ./manifest/ directory using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_PUSH_MANIFEST_FILE} | bash -s -- ${PUSH_PARAMS}
endif

PHONY: copy-manifests-to-community-operators
## Copies the ./manifest/ directory into community-operators/codeready-toolchain-operator/ directory
copy-manifests-to-community-operators:
ifeq ("$(wildcard $(COMMUNITY_OPERATORS_DIR))","")
	$(error The directory ${COMMUNITY_OPERATORS_DIR} is not available. Clone the repository and pull the latest changes, then run the target again.)
endif
	rm -rf ${COMMUNITY_OPERATORS_DIR}/*
	cp -r manifests/* ${COMMUNITY_OPERATORS_DIR}/

PHONY: delete-release-manifest-from-os
## Deletes CatalogSource 'source-codeready-toolchain-operator' and ConfigMap 'cm-codeready-toolchain-operator' from OpenShift
delete-release-manifest-from-os:
	oc delete catalogsource source-codeready-toolchain-operator -n openshift-marketplace 2>/dev/null || true
	oc delete configmap cm-codeready-toolchain-operator -n openshift-marketplace 2>/dev/null || true

.PHONY: add-release-manifests-to-os
## Creates ServiceCatalog with a ConfigMap that contains operator CSV and all CRDs and image location set to current OS registry
add-release-manifests-to-os:
	$(eval CREATE_PARAMS = -crds ./deploy/crds -csvs ./manifests/ -pf ./manifests/codeready-toolchain-operator.package.yaml -hd /tmp/hack_deploy_crt-operator_${DATE_SUFFIX} -on codeready-toolchain-operator)
ifneq ("$(wildcard ../api/$(PATH_TO_CREATE_HACK_FILE))","")
	@echo "adding release manifests from ./manifest/ directory to OpenShift using script from local api repo..."
	../api/${PATH_TO_CREATE_HACK_FILE} ${CREATE_PARAMS}
else
	@echo "adding release manifests from ./manifest/ directory to OpenShift using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_CREATE_HACK_FILE} | bash -s -- ${CREATE_PARAMS}
endif
	cat /tmp/hack_deploy_crt-operator_${DATE_SUFFIX}/deploy_csv.yaml | oc apply -f -

.PHONY: push-to-quay-nightly
## Creates a new version of CSV and pushes it to quay
push-to-quay-nightly:
	$(eval PUSH_PARAMS = -pr ../toolchain-operator/ -on codeready-toolchain-operator -qn ${QUAY_NAMESPACE} -ch nightly)
ifneq ("$(wildcard ../api/$(PATH_TO_PUSH_NIGHTLY_FILE))","")
	@echo "pushing to quay in nightly channel using script from local api repo..."
	../api/${PATH_TO_PUSH_NIGHTLY_FILE} ${PUSH_PARAMS}
else
	@echo "pushing to quay in nightly channel using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_PUSH_NIGHTLY_FILE} | bash -s -- ${PUSH_PARAMS}
endif