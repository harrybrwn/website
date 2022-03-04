DATE=$(shell date '+%a, %d %b %Y %H:%M:%S %Z')
ENV=production

build:
	sh scripts/build.sh

test: test-ts test-go

lint: lint-go

clean:
	$(RM) -r bin .testing .build
	$(RM) test-cover files/resume.pdf files/resume.log files/resume.aux
	yarn clean

coverage: coverage-ts coverage-go

deep-clean:
	$(RM) -r internal/mocks \
		$(shell find . -name '.pytest_cache' -type d) \
		$(shell find . -name '__pycache__' -type d)

test-go:
	@mkdir -p .testing
	go generate ./...
	go test -tags ci ./... -covermode=atomic -coverprofile=.testing/coverprofile.txt
	go tool cover -html=.testing/coverprofile.txt -o .testing/coverage.html
	@#x-www-browser .testing/coverage.html

test-ts:
	yarn test

.PHONY: coverage-go coverage-ts
coverage-go:
	x-www-browser .testing/coverage.html

coverage-ts:
	yarn coverage

lint-go:
	go vet -tags ci ./...
	golangci-lint run

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

.PHONY: functional-setup build-functional
build-functional:
	docker-compose -f docker-compose.test.yml build

functional-setup:
	docker-compose -f docker-compose.yml -f docker-compose.test.yml down
	docker-compose -f docker-compose.yml -f docker-compose.test.yml build
	docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d db redis web

functional:
	docker-compose -f docker-compose.yml -f docker-compose.test.yml run --rm tests scripts/functional-tests.sh
