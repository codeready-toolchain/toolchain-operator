############################################################
#
# (local) Tests
#
############################################################

.PHONY: test
## runs the tests without coverage and excluding E2E tests
test: generate
	@echo "running the tests without coverage and excluding E2E tests..."
	$(Q)go test ${V_FLAG} -race $(shell go list ./... | grep -v /test/e2e) -failfast


############################################################
#
# OpenShift CI Tests with Coverage
#
############################################################

# Output directory for coverage information
COV_DIR = $(OUT_DIR)/coverage

.PHONY: test-with-coverage
## runs the tests with coverage
test-with-coverage: 
	@echo "running the tests with coverage..."
	@-mkdir -p $(COV_DIR)
	@-rm $(COV_DIR)/coverage.txt
	$(Q)go test -vet off ${V_FLAG} $(shell go list ./... | grep -v /test/e2e) -coverprofile=$(COV_DIR)/coverage.txt -covermode=atomic ./...

.PHONY: upload-codecov-report
# Uploads the test coverage reports to codecov.io. 
# DO NOT USE LOCALLY: must only be called by OpenShift CI when processing new PR and when a PR is merged! 
upload-codecov-report: 
	# Upload coverage to codecov.io. Since we don't run on a supported CI platform (Jenkins, Travis-ci, etc.), 
	# we need to provide the PR metadata explicitely using env vars used coming from https://github.com/openshift/test-infra/blob/master/prow/jobs.md#job-environment-variables
	# 
	# Also: not using the `-F unittests` flag for now as it's temporarily disabled in the codecov UI 
	# (see https://docs.codecov.io/docs/flags#section-flags-in-the-codecov-ui)
	env
ifneq ($(PR_COMMIT), null)
	@echo "uploading test coverage report for pull-request #$(PULL_NUMBER)..."
	bash <(curl -s https://codecov.io/bash) \
		-t $(CODECOV_TOKEN) \
		-f $(COV_DIR)/coverage.txt \
		-C $(PR_COMMIT) \
		-r $(REPO_OWNER)/$(REPO_NAME) \
		-P $(PULL_NUMBER) \
		-Z
else
	@echo "uploading test coverage report after PR was merged..."
	bash <(curl -s https://codecov.io/bash) \
		-t $(CODECOV_TOKEN) \
		-f $(COV_DIR)/coverage.txt \
		-C $(BASE_COMMIT) \
		-r $(REPO_OWNER)/$(REPO_NAME) \
		-Z
endif

CODECOV_TOKEN := "b4bc232f-a825-4dc2-add1-5ab6e896b0a4"
REPO_OWNER := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].org')
REPO_NAME := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].repo')
BASE_COMMIT := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].base_sha')
PR_COMMIT := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].pulls[0].sha')
PULL_NUMBER := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].pulls[0].number')

TOOLCHAIN_NS := openshift-operators

###########################################################
#
# End-to-end Tests
#
###########################################################

DATE_SUFFIX := $(shell date +'%s')

IS_OS_3 := $(shell curl -k -XGET -H "Authorization: Bearer $(shell oc whoami -t 2>/dev/null)" $(shell oc config view --minify -o jsonpath='{.clusters[0].cluster.server}')/version/openshift 2>/dev/null | grep paths)
IS_CRC := $(shell oc config view --minify -o jsonpath='{.clusters[0].cluster.server}' 2>&1 | grep crc)
IS_OS_CI := $(OPENSHIFT_BUILD_NAMESPACE)
IS_KUBE_ADMIN := $(shell oc whoami | grep "kube:admin")

.PHONY: test-e2e-keep-resources
test-e2e-keep-resources: e2e-setup e2e-run

.PHONY: test-e2e
test-e2e: test-e2e-keep-resources clean-e2e-resources

.PHONY: e2e-run
e2e-run:
	operator-sdk test local ./test/e2e --no-setup --namespace $(TOOLCHAIN_NS) --verbose --go-test-flags "-timeout=15m" || \
	($(MAKE) print-logs TOOLCHAIN_NS=${TOOLCHAIN_NS} && exit 1)

.PHONY: print-logs
print-logs:
	@echo "=====================================================================================" &
	@echo "============================== Toolchain cluster logs ==============================="
	@echo "====================================================================================="
	@oc logs deployment.apps/toolchain-operator --namespace $(TOOLCHAIN_NS)
	@echo "====================================================================================="

.PHONY: e2e-setup
e2e-setup: build-image
	oc project $(TOOLCHAIN_NS) 1>/dev/null
ifneq ($(IS_OS_3),)
	oc apply -f ./deploy/service_account.yaml
	oc apply -f ./deploy/role.yaml
	oc apply -f ./deploy/role_binding.yaml
	oc apply -f ./deploy/cluster_role.yaml
	oc apply -f ./deploy/cluster_role_binding.yaml
	sed -e 's|REPLACE_NAMESPACE|${TOOLCHAIN_NS}|g' ./deploy/cluster_role_binding.yaml | oc apply -f -
	oc apply -f deploy/crds
	sed -e 's|REPLACE_IMAGE|${IMAGE_NAME}|g' ./deploy/operator.yaml  | oc apply -f -
else
	# it is not using OS 3 so we will install operator via CSV
	$(eval REPO_NAME := ${GO_PACKAGE_REPO_NAME})
	sed -e 's|REPLACE_IMAGE|${IMAGE_NAME}|g;s|^  name: .*|&-${DATE_SUFFIX}|;s|^  configMap: .*|&-${DATE_SUFFIX}|' ./hack/deploy_csv.yaml > /tmp/${REPO_NAME}_deploy_csv_${DATE_SUFFIX}.yaml
	cat /tmp/${REPO_NAME}_deploy_csv_${DATE_SUFFIX}.yaml | oc apply -f -
	sed -e 's|REPLACE_NAMESPACE|${TOOLCHAIN_NS}|g;s|^  source: .*|&-${DATE_SUFFIX}|' ./hack/install_operator.yaml > /tmp/${REPO_NAME}_install_operator_${DATE_SUFFIX}.yaml
	cat /tmp/${REPO_NAME}_install_operator_${DATE_SUFFIX}.yaml | oc apply -f -
	while [[ -z `oc get sa ${REPO_NAME} -n ${TOOLCHAIN_NS} 2>/dev/null` ]]; do \
		if [[ $${NEXT_WAIT_TIME} -eq 300 ]]; then \
		   CATALOGSOURCE_NAME=`oc get catalogsource --output=name -n openshift-marketplace | grep "${REPO_NAME}.*${DATE_SUFFIX}"`; \
		   SUBSCRIPTION_NAME=`oc get subscription --output=name -n ${TOOLCHAIN_NS} | grep "${REPO_NAME}"`; \
		   echo "reached timeout of waiting for ServiceAccount ${REPO_NAME} to be available in namespace ${TOOLCHAIN_NS} - see following info for debugging:"; \
		   echo "================================ CatalogSource =================================="; \
		   oc get $${CATALOGSOURCE_NAME} -n openshift-marketplace -o yaml; \
		   echo "================================ CatalogSource Pod Logs =================================="; \
		   oc logs `oc get pods -l "olm.catalogSource=$${CATALOGSOURCE_NAME#*/}" -n openshift-marketplace -o name` -n openshift-marketplace; \
		   echo "================================ Subscription =================================="; \
		   oc get $${SUBSCRIPTION_NAME} -n ${TOOLCHAIN_NS} -o yaml; \
		   $(MAKE) print-logs TOOLCHAIN_NS=${TOOLCHAIN_NS}; \
		   exit 1; \
		fi; \
		echo "$$(( NEXT_WAIT_TIME++ )). attempt of waiting for ServiceAccount ${REPO_NAME} in namespace ${TOOLCHAIN_NS}"; \
		sleep 1; \
	done
endif

.PHONY: build-image
build-image:
ifneq ($(IS_OS_3),)
	$(info logging as system:admin")
	$(shell echo "oc login -u system:admin")
	$(eval IMAGE_NAME := docker.io/${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}:${GIT_COMMIT_ID_SHORT})
	$(MAKE) docker-image IMAGE=${IMAGE_NAME}
else ifneq ($(IS_OS_CI),)
	$(eval IMAGE_NAME := registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:toolchain-operator)
else
	# For OpenShift-4
	$(eval IMAGE_NAME := quay.io/${QUAY_NAMESPACE}/${GO_PACKAGE_REPO_NAME}:${DATE_SUFFIX})
	$(MAKE) docker-push IMAGE=${IMAGE_NAME}
endif

.PHONY: clean-e2e-resources
clean-e2e-resources:
	oc get catalogsource --output=name -n openshift-marketplace | grep "toolchain-operator" | xargs --no-run-if-empty oc delete -n openshift-marketplace
	oc get subscription --output=name -n ${TOOLCHAIN_NS} |  grep "toolchain-operator" | xargs --no-run-if-empty oc delete -n ${TOOLCHAIN_NS}
	oc get subscription --output=name -n openshift-operators |  grep "openshift-pipelines-operator" | xargs --no-run-if-empty oc delete -n openshift-operators
	oc delete project toolchain-che --timeout=10s 2 > /dev/null || true
