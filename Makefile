DATE=$(shell date '+%a, %d %b %Y %H:%M:%S %Z')
ENV=production
TESTCACHE=.cache/test
BUILDCACHE=.cache/build

build:
	sh scripts/build.sh

test: test-ts test-go

lint: lint-go

clean:
	$(RM) -r bin .cache .pytest_cache .cache \
		$(shell find . -name '.pytest_cache' -type d) \
		$(shell find . -name '__pycache__' -type d) \
		test-cover files/resume.pdf files/resume.log files/resume.aux
	yarn clean

coverage: coverage-ts coverage-go

deep-clean: clean
	$(RM) -r internal/mocks node_modules

test-go:
	@mkdir -p .cache/test
	go generate ./...
	go test -tags ci ./... -covermode=atomic -coverprofile=.cache/test/coverprofile.txt
	go tool cover -html=.cache/test/coverprofile.txt -o .cache/test/coverage.html
	@#x-www-browser .cache/test/coverage.html

test-ts:
	yarn test

.PHONY: coverage-go coverage-ts
coverage-go:
	x-www-browser .cache/test/coverage.html

coverage-ts:
	yarn coverage

lint-go:
	go vet -tags ci ./...
	golangci-lint run --config ./config/golangci.yml

TOOLS=user-gen pwhash key-gen

tools:
	@mkdir -p ./bin
	@for tool in $(TOOLS); do \
		go build -trimpath -ldflags "-s -w" -o ./bin/$$tool ./cmd/$$tool; \
	done

resume:
	docker container run --rm -it -v $(shell pwd):/app latex \
		pdflatex \
		--output-directory=/app/files \
		/app/files/resume.tex

.PHONY: latex-image
latex-image:
	docker image build -t latex -f config/docker/Dockerfile.latex .

blog: build/blog
.PHONY: blog

build/blog: blog/resources/remora.svg
	hugo --environment $(ENV)

blog/resources/remora.svg: diagrams/remora.svg
	cp $< $@

diagrams/remora.svg: diagrams/remora.drawio
	./scripts/diagrams.svg

.PHONY: build run test clean deep-clean test-go test-ts resume tools

functional-build:
	scripts/functional.sh build

functional-setup:
	scripts/functional.sh build
	scripts/functional.sh setup

functional-run:
	scripts/functional.sh run

functional-stop:
	scripts/functional.sh stop

functional: functional-setup functional-run functional-stop

.PHONY: functional functional-setup functional-run functional-run functional-build
