TF ?= terraform
TF_ARGS ?=
TF_LOCK_TIMEOUT ?= 30s

# You have to provide the cluster name for some commands
OKE_CLUSTER ?= 

check_defined = $(if $(value OKE_CLUSTER),,$(error OKE_CLUSTER is not set))

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
cluster-init: ## Initialize Terraform's state and download necessary providers.
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
	rm -f plan.out

cluster-destroy: ## Delete Terraform resources.
	@:$(call check_defined, OKE_CLUSTER, cluster name)
	$(TF) -chdir=./cluster \
		destroy $(TF_ARGS) \
		-lock-timeout=$(TF_LOCK_TIMEOUT) \
		-var-file=./tfvars/$(OKE_CLUSTER).tfvars

.PHONY: output
cluster-output: ## Print Terraform output.
	@$(TF) -chdir=./cluster output -json

.PHONY: clean
clean: ## Clean up Terraform cache and local state.
	rm -rf ./clusters/.terraform ./clusters/.terraform.lock.hcl ./clusters/plan.out
