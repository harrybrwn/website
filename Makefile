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

clean-mocks:
	$(RM) -r internal/mocks

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

.PHONY: build run test clean clean-mocks test-go test-ts resume tools

.PHONY: functional-setup build-functional
build-functional:
	docker-compose -f docker-compose.test.yml build
functional-setup:
	docker-compose -f docker-compose.test.yml -f docker-compose.yml down
	docker-compose -f docker-compose.test.yml -f docker-compose.yml up -d db redis web
	docker-compose -f docker-compose.test.yml -f docker-compose.yml run --rm tests scripts/functional-setup.sh
	docker-compose -f docker-compose.test.yml -f docker-compose.yml logs -f

