.DEFAULT_GOAL := help

.PHONY: gen
gen: ## Generate from controller-gen
	@go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.1
	@$(shell go env GOPATH)/bin/controller-gen paths="./..." object crd:trivialVersions=true output:crd:artifacts:config=manifests/crd

.PHONY: dev
dev: ## Run skaffold
	@skaffold dev

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
