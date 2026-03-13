TF ?= terraform
TF_ARGS ?=
TF_LOCK_TIMEOUT ?= 30s

# You have to provide the cluster name for some commands
OKE_CLUSTER ?= 

# You have to provide the bucket group name for some commands
BUCKETS ?= 

check_defined = $(if $(value OKE_CLUSTER),,$(error OKE_CLUSTER is not set))
check_buckets_defined = $(if $(value BUCKETS),,$(error BUCKETS is not set))

##@ Helpers:

.PHONY: help
help: ## Display this help.
	awk \
		-v "col=${COLOR}" -v "nocol=${NOCOLOR}" \
		' \
		BEGIN { \
			FS = ":.*##" ; \
			printf "\nUsage:\n  make %s<target>%s\n", col, nocol \
		} \
		/^[a-zA-Z_-]+:.*?##/ { \
			printf "  %s%-15s%s %s\n", col, $$1, nocol, $$2 \
		} \
		/^##@/ { \
			printf "\n%s%s%s\n", col, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)

### Clusters
cluster-init: ## Initialize Terraform's state and download necessary providers for OKE.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	$(TF) -chdir=./cluster \
		init \
		-backend-config=./tfbackends/$(OKE_CLUSTER).tfbackend

cluster-plan: ## Present plan for creating/updating Terraform resources.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	$(TF) -chdir=./cluster \
		plan $(TF_ARGS) \
		-out=plan.out \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(OKE_CLUSTER).tfvars

cluster-apply: ## Create/Update Terraform resources.
	$(TF) -chdir=./cluster \
		apply $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		"plan.out"
	rm -f ./cluster/plan.out

cluster-destroy: ## Delete Terraform resources.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	$(TF) -chdir=./cluster \
		destroy $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(OKE_CLUSTER).tfvars

.PHONY: output
cluster-output: ## Print Terraform output.
	@$(TF) -chdir=./cluster output -json

### Buckets
buckets-init: ## Initialize Terraform's state and download necessary providers for buckets.
	@:$(call check_buckets_defined, BUCKETS, bucket group name)
	$(TF) -chdir=./buckets \
		init \
		-backend-config=./tfbackends/$(BUCKETS).tfbackend

buckets-plan: ## Present plan for creating/updating bucket resources.
	@:$(call check_buckets_defined, BUCKETS, bucket group name)
	$(TF) -chdir=./buckets \
		plan $(TF_ARGS) \
		-out=plan.out \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(BUCKETS).tfvars

buckets-apply: ## Create/Update bucket resources.
	$(TF) -chdir=./buckets \
		apply $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		"plan.out"
	rm -f ./buckets/plan.out

buckets-destroy: ## Delete bucket resources.
	@:$(call check_buckets_defined, BUCKETS, bucket group name)
	$(TF) -chdir=./buckets \
		destroy $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(BUCKETS).tfvars

buckets-output: ## Print bucket Terraform output.
	@$(TF) -chdir=./buckets output -json

### Clean up
.PHONY: clean clean-cluster clean-buckets
clean: clean-cluster clean-buckets ## Clean up all Terraform cache and local state.

clean-cluster: ## Clean up cluster Terraform cache and local state.
	rm -rf ./cluster/.terraform ./cluster/.terraform.lock.hcl ./cluster/plan.out

clean-buckets: ## Clean up buckets Terraform cache and local state.
	rm -rf ./buckets/.terraform ./buckets/.terraform.lock.hcl ./buckets/plan.out
