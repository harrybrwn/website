DATE=$(shell date '+%a, %d %b %Y %H:%M:%S %Z')
ENV=?production
TESTCACHE=.cache/test
BUILDCACHE=.cache/build

help:
	@echo Makefile for hrry.me
	@echo
	@echo 'Targets'
	@echo '  help    print this help message'
	@echo '  tools   build the tooling'
	@echo '  init    initialize the environment for builds and management'
	@echo
	@echo 'Variables'
	@echo '  ENV     environment (default: "production")'

init:
	terraform -chdir=terraform init
	terraform/projects/homelab/tf init
	yarn

build: tools
	bin/bake --local --load

test: test-ts test-go

.PHONY: help init build test

lint: lint-go lint-rs lint-sh

clean:
	$(RM) -r .cache .pytest_cache .cache \
		test-cover files/resume.pdf files/resume.log files/resume.aux
	yarn clean
	$(RM) result result-man

coverage: coverage-ts coverage-go

deep-clean: clean
	$(RM) -r internal/mocks \
		$(shell find . -name 'node_modules' -type d)  \
		$(shell find . -name 'yarn-error.log')
	sudo $(RM) -r \
		$(shell find . -name '.pytest_cache' -type d) \
		$(shell find . -name '__pycache__' -type d)

test-go:
	@mkdir -p .cache/test
	go generate ./...
	go test -tags ci ./... -covermode=atomic -coverprofile=.cache/test/coverprofile.txt
	go tool cover -html=.cache/test/coverprofile.txt -o .cache/test/coverage.html
	@#x-www-browser .cache/test/coverage.html

.PHONY: coverage-go coverage-ts
coverage-go:
	x-www-browser .cache/test/coverage.html

lint-go:
	go vet -tags ci ./...
	golangci-lint run --config ./config/golangci.yml

lint-sh:
	@shellcheck -x \
		$(shell find ./scripts/ -name '*.sh' -type f) \
		$(shell find ./scripts/tools -type f)

lint-rs:
	cargo clippy

lint-yml:
	yamllint -c config/yamllint.yml .

lint-ansible:
	bin/ansible-lint -c config/ansible/ansible-lint.yml

scripts:
	@mkdir -p bin
	ln -sf ../scripts/functional.sh bin/functional
	@for s in hydra bake k8s tootctl; do \
		echo ln -sf "../scripts/tools/$$s" "bin/$$s"; \
		ln -sf "../scripts/tools/$$s" "bin/$$s"; \
	done
	ln -sf ../scripts/infra/ansible bin/ansible
	@for s in playbook inventory config galaxy test pull console connection vault lint; do \
		echo ln -sf ../scripts/infra/ansible bin/ansible-$$s; \
		ln -sf ../scripts/infra/ansible bin/ansible-$$s; \
	done
	ln -sf $$HOME/dev/bluesky/pds/scripts/pdsadmin.sh bin/pdsadmin

tools: scripts bin/lab
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w" -o bin/provision ./cmd/provision
	go build -trimpath -ldflags "-s -w" -o bin/user-gen ./cmd/tools/user-gen
	go build -trimpath -ldflags "-s -w" -o bin/mail ./cmd/tools/mail
	docker compose -f config/docker-compose.tools.yml --project-directory $(shell pwd) build ansible

.PHONY: tools scripts

bin/lab: $(shell find ./cmd/tools/lab -type f)
	@mkdir -p bin
	(cd cmd/tools/lab && \
		go build -trimpath -ldflags "-s -w" -o ../../../bin/lab .)

# https://dev.maxmind.com/geoip/updating-databases?lang=en
geoip:
	scripts/data/geoipupdate.sh

.PHONY: run clean deep-clean test-go test-ts
.PHONY: lint-go lint-sh lint-rs lint-yml lint-ansible

k8s:
	make -C config/k8s all

helm:
	make -C config/helm build

.PHONY: k8s helm

vpn:
	pushd terraform/projects/vpn
	tofu apply
	./import.sh "$(shell tofu -chdir=terraform/projects/vpn output --raw config)"
	popd

vpn-destroy:
	pushd terraform/projects/vpn
	tofu destroy
	popd

.PHONY: vpn vpn-destroy

dist:
	mkdir dist

gomod2nix.toml:
	nix develop --command gomod2nix generate
