VERSION_TAG?=1.25.3
MODULES=geoip geoip2 headers-more opentracing otel ndk encrypted-session

image: docker/Dockerfile
	docker image build \
		--build-arg "ENABLED_MODULES=$(MODULES)" \
		--build-arg "NGINX_FROM_IMAGE=nginx:$(VERSION_TAG)-alpine" \
		--file $< \
		--tag nginx-test:latest \
		.

buildx: docker/Dockerfile
	docker buildx build \
		--build-arg "ENABLED_MODULES=$(MODULES)" \
		--build-arg "NGINX_FROM_IMAGE=nginx:$(VERSION_TAG)-alpine" \
		--file $< \
		--platform "linux/amd64,linux/arm/v7,linux/arm64" \
		--tag nginx-test:latest \
		.

.PHONY: image buildx

docker/Dockerfile:
	@mkdir -p docker/
	curl -sSL -o $@ https://raw.githubusercontent.com/nginxinc/docker-nginx/$(VERSION_TAG)/modules/Dockerfile.alpine
