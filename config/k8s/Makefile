env?=dev
APP_MANIFESTS=$(shell find ./app ./common -type f -name '*.yml' -o -name '*.yaml')
DEV_MANIFESTS=$(shell find ./dev -type f -name '*.yml' -o -name '*.yaml')
STG_MANIFESTS=$(shell find ./stg -type f -name '*.yml' -o -name '*.yaml')
PRD_MANIFESTS=$(shell find ./prd -type f -name '*.yml' -o -name '*.yaml')

help:
	@echo "Usage"
	@echo "  make <target> [env=<dev|stg|prd>]"
	@echo
	@echo "Variables"
	@echo "  env [dev|stg|prd]   Choose the environment (default 'dev')"
	@echo
	@echo "Targets"
	@echo "  help     Print this help message"
	@echo "  build    Build kubernetes manifests"
	@echo "  lint     Lint manifests"
	@echo "  clean    Remove build files"

build: dist/$(env)
all: dist/dev dist/stg dist/prd

lint: build
	# kubectl kustomize config/k8s/dev | kube-score score -
	kubeval -d . \
	  --ignored-path-patterns 'kustomization.yml,kustomization.yaml,registry/config.yml,prd/patches/,stg/patches/,k3d.yml,generate/test-config.yml' \
	  --ignore-missing-schemas
	# kubectl kustomize ./dev | kubeval --ignore-missing-schemas
	# kubectl kustomize ./stg | kubeval --ignore-missing-schemas
	# kubectl kustomize ./prd | kubeval --ignore-missing-schemas

clean:
	$(RM) -r dist

.PHONY: build lint

dist/dev: $(APP_MANIFESTS) $(DEV_MANIFESTS)
	@mkdir -p $@
	kubectl kustomize ./dev -o $@/manifest.yml

dist/stg: $(APP_MANIFESTS) $(STG_MANIFESTS)
	@mkdir -p $@
	kubectl kustomize ./stg -o $@/manifest.yml

dist/prd: $(APP_MANIFESTS) $(PRD_MANIFESTS)
	@mkdir -p $@
	kubectl kustomize ./prd -o $@/manifest.yml
