CSV_VERSION_TO_GENERATE := 0.1.1
COMMUNITY_OPERATORS_DIR=../../operator-framework/community-operators/community-operators/codeready-toolchain-operator

PATH_TO_GENERATE_FILE=../api/scripts/olm-catalog-generate.sh

PHONY: create-release-manifest
create-release-manifest:
	$(eval CREATE_PARAMS = -pr ../toolchain-operator -on codeready-toolchain-operator  --next-version ${CSV_VERSION_TO_GENERATE})
ifneq ("$(wildcard $(PATH_TO_GENERATE_FILE))","")
	@echo "creating release manifest in ./manifest/ directory using script from local api repo..."
	../api/scripts/create-release-bundle.sh ${CREATE_PARAMS}
else
	@echo "creating release manifest in ./manifest/ directory using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/scripts/create-release-bundle.sh | bash -s -- ${CREATE_PARAMS}
endif

PHONY: push-latest-release-manifest
push-latest-release-manifest:
	$(eval PUSH_PARAMS = -pr ../toolchain-operator -on codeready-toolchain-operator)
ifneq ("$(wildcard $(PATH_TO_GENERATE_FILE))","")
	@echo "pushing the latest release manifest from ./manifest/ directory using script from local api repo..."
	../api/scripts/push-to-quay-manifests.sh ${PUSH_PARAMS}
else
	@echo "ushing the latest release manifest from ./manifest/ directory using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/scripts/push-to-quay-manifests.sh | bash -s -- ${PUSH_PARAMS}
endif

PHONY: copy-manifests-to-community-operators
copy-manifests-to-community-operators:
ifeq ("$(wildcard $(COMMUNITY_OPERATORS_DIR))","")
	$(error The directory ${COMMUNITY_OPERATORS_DIR} is not available. Clone the repository and pull the latest changes, then run the target again.)
endif
	rm -rf ${COMMUNITY_OPERATORS_DIR}/*
	cp -r manifests/* ${COMMUNITY_OPERATORS_DIR}/

PHONY: delete-release-manifest-from-os
delete-release-manifest-from-os:
	oc delete catalogsource source-codeready-toolchain-operator -n openshift-marketplace 2>/dev/null || true
	oc delete configmap cm-codeready-toolchain-operator -n openshift-marketplace 2>/dev/null || true

.PHONY: add-release-manifest-to-os
## Creates ServiceCatalog with a ConfigMap that contains operator CSV and all CRDs and image location set to current OS registry
add-release-manifest-to-os:
	cat /tmp/hack_deploy_codeready-toolchain-operator_${CSV_VERSION_TO_GENERATE}/deploy_csv.yaml | oc apply -f -