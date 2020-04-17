.DEFAULT_GOAL := help

.PHONY: gen
gen: ## Generate from controller-gen
	@go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.9
	@$(shell go env GOPATH)/bin/controller-gen paths="./..." object crd:crdVersions=v1,trivialVersions=true output:crd:artifacts:config=manifests/crd

.PHONY: test
test: ## Test
	@go test ./... -race -bench . -benchmem -trimpath -cover

.PHONY: lint
lint: ## Lint
	@go get golang.org/x/tools/cmd/goimports@5916a50
	@go get github.com/instrumenta/kubeval@0.14.0
	@for d in $(shell go list -f {{.Dir}} ./...); do $(shell go env GOPATH)/bin/goimports -w $$d/*.go; done
	@docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.21.0 golangci-lint run --fix
	@$(shell go env GOPATH)/bin/kubeval --strict --ignore-missing-schemas manifests/cluster_role.yaml manifests/cluster_role_binding.yaml manifests/deployment.yaml manifests/pod_disruption_budget.yaml manifests/role.yaml manifests/role_binding.yaml manifests/service_account.yaml

.PHONY: dev
dev: ## Run skaffold
	@skaffold dev

.PHONY: help
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
