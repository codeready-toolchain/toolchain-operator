CSV_VERSION_TO_GENERATE := 0.1.4
DATE_SUFFIX := $(shell date +'%d%H%M%S')
COMMUNITY_OPERATORS_DIR=../../operator-framework/community-operators/community-operators/codeready-toolchain-operator

# For final releases
PATH_TO_CREATE_RELEASE_FILE= scripts/create-release-bundle.sh
PATH_TO_CREATE_HACK_FILE= scripts/generate-deploy-hack.sh

# For CD
PATH_TO_CD_GENERATE_FILE=scripts/generate-cd-release-manifests.sh
PATH_TO_PUSH_APP_FILE=scripts/push-manifests-as-app.sh
PATH_TO_BUNDLE_FILE=scripts/push-bundle-and-index-image.sh
PATH_TO_RECOVERY_FILE=scripts/recover-operator-dir.sh
PATH_TO_OLM_GENERATE_FILE=scripts/olm-catalog-generate.sh

TMP_DIR?=/tmp
IMAGE_BUILDER?=docker
INDEX_IMAGE?=toolchain-operator-index

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
push-to-quay-nightly: generate-cd-release-manifests push-manifests-as-app recover-operator-dir

.PHONY: push-to-quay-staging
## Creates a new version of operator bundle, adds it into an index and pushes it to quay
push-to-quay-staging: generate-cd-release-manifests push-bundle-and-index-image recover-operator-dir

.PHONY: generate-cd-release-manifests
## Generates a new version of operator manifests
generate-cd-release-manifests:
	$(eval CD_GENERATE_PARAMS = -pr ../toolchain-operator/ -on codeready-toolchain-operator -qn ${QUAY_NAMESPACE} -td ${TMP_DIR})
ifneq ("$(wildcard ../api/$(PATH_TO_CD_GENERATE_FILE))","")
	@echo "generating manifests for CD using script from local api repo..."
	../api/${PATH_TO_CD_GENERATE_FILE} ${CD_GENERATE_PARAMS}
else
	@echo "generating manifests for CD using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_CD_GENERATE_FILE} | bash -s -- ${CD_GENERATE_PARAMS}
endif

.PHONY: push-manifests-as-app
## Pushes generated manifests as an application to quay
push-manifests-as-app:
	$(eval PUSH_APP_PARAMS = -pr ../toolchain-operator/ -on codeready-toolchain-operator -qn ${QUAY_NAMESPACE} -ch nightly -td ${TMP_DIR})
ifneq ("$(wildcard ../api/$(PATH_TO_PUSH_APP_FILE))","")
	@echo "pushing to quay in nightly channel using script from local api repo..."
	../api/${PATH_TO_PUSH_APP_FILE} ${PUSH_APP_PARAMS}
else
	@echo "pushing to quay in nightly channel using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_PUSH_APP_FILE} | bash -s -- ${PUSH_APP_PARAMS}
endif

.PHONY: push-bundle-and-index-image
## Pushes generated manifests as a bundle image to quay and adds is to the image index
push-bundle-and-index-image:
	$(eval PUSH_BUNDLE_PARAMS = -pr ../toolchain-operator/ -on codeready-toolchain-operator -qn ${QUAY_NAMESPACE} -ch staging -td ${TMP_DIR} -ib ${IMAGE_BUILDER} -im ${INDEX_IMAGE})
ifneq ("$(wildcard ../api/$(PATH_TO_BUNDLE_FILE))","")
	@echo "pushing to quay in staging channel using script from local api repo..."
	../api/${PATH_TO_BUNDLE_FILE} ${PUSH_BUNDLE_PARAMS}
else
	@echo "pushing to quay in staging channel using script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_BUNDLE_FILE} | bash -s -- ${PUSH_BUNDLE_PARAMS}
endif

.PHONY: recover-operator-dir
## Recovers the operator directory from the backup folder
recover-operator-dir:
	$(eval RECOVERY_PARAMS = -pr ../toolchain-operator/ -td ${TMP_DIR})
ifneq ("$(wildcard ../api/$(PATH_TO_RECOVERY_FILE))","")
	@echo "recovering the operator directory from the backup folder using script from local api repo..."
	../api/${PATH_TO_RECOVERY_FILE} ${RECOVERY_PARAMS}
else
	@echo "recovering the operator directory from the backup folder script from GH api repo (using latest version in master)..."
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/api/master/${PATH_TO_RECOVERY_FILE} | bash -s -- ${RECOVERY_PARAMS}
endif

