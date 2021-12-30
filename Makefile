DATE=$(shell date '+%a, %d %b %Y %H:%M:%S %Z')
ENV=production

build:
	go generate
	go build             \
		-trimpath        \
		-ldflags "-s -w" \
		-o bin/harrybrown.com

run: build
	@bin/harrybrown.com

test:
	go test ./... -coverprofile=test-cover
	go tool cover -html=test-cover

clean:
	$(RM) bin -r
	$(RM) test.html test-cover
	yarn clean

.PHONY: build clean

blog: build/blog
.PHONY: blog

build/blog: blog/resources/remora.svg
	hugo --environment $(ENV)

blog/resources/remora.svg: diagrams/remora.svg
	cp $< $@

diagrams/remora.svg: diagrams/remora.drawio
	./scripts/diagrams.svg

