# This Makefile is the contributor entrypoint for generation, testing, packaging,
# and local execution. It exists so maintainers can use consistent commands in CI
# and on workstations without needing prior repository-specific context.

IMG             ?= ghcr.io/sandy001-kki/shukra-operator:latest
CLI_BIN         ?= shukra
CHART_VERSION   ?= 0.2.3
ENVTEST_VERSION ?= release-0.17
KUSTOMIZE       ?= kustomize
CONTROLLER_GEN  ?= controller-gen
GOLANGCI_LINT   ?= golangci-lint
CRD_REF_DOCS    ?= crd-ref-docs
ENVTEST_ASSETS  ?= $(shell setup-envtest use $(ENVTEST_VERSION) -p path 2>/dev/null)

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile=hack/boilerplate.go.txt paths="./..."

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) \
		rbac:roleName=manager-role \
		crd \
		webhook \
		paths="./..." \
		output:crd:artifacts:config=config/crd/bases

.PHONY: install
install:
	kubectl apply -f config/crd/bases

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/bases --ignore-not-found=true

.PHONY: run
run:
	go run ./cmd/main.go --leader-elect=true --max-concurrent-reconciles=5

.PHONY: cli-build
cli-build:
	go build -o bin/$(CLI_BIN) ./cmd/shukra

.PHONY: test
test:
	KUBEBUILDER_ASSETS="$(ENVTEST_ASSETS)" go test ./controllers/... ./webhooks/... -coverprofile cover.out

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push:
	docker push $(IMG)

.PHONY: helm-package
helm-package:
	helm package charts/shukra-operator --version $(CHART_VERSION) --app-version $(CHART_VERSION)

.PHONY: docs-generate
docs-generate:
	$(CRD_REF_DOCS) --source-path=./api --renderer=markdown --output-path=./docs/api.md

.PHONY: bootstrap-local
bootstrap-local:
	powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1

.PHONY: ai-dataset
ai-dataset:
	powershell -ExecutionPolicy Bypass -File .\hack\prepare-ai-dataset.ps1

.PHONY: ai-eval
ai-eval:
	powershell -ExecutionPolicy Bypass -File .\hack\evaluate-ai-readiness.ps1
