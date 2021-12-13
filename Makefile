DATE=$(shell date '+%a, %d %b %Y %H:%M:%S %Z')

build:
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

.PHONY: build clean
