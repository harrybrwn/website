VERSION_TAG?=1.25.3
MODULES=geoip geoip2 headers-more opentracing otel ndk encrypted-session

help:
	@echo "Nginx Custom Build"
	@echo
	@echo "Usage"
	@echo "  make <target>"
	@echo
	@echo "Targets"
	@echo "  generate   generate files"
	@echo "  image      build the docker image"
	@echo "  buildx     build multi-arch docker image"

generate: generate-cloudflare

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

generate-cloudflare:
	@rm snippets/cloudflare-allow.conf
	@for u in https://www.cloudflare.com/ips-v4 https://www.cloudflare.com/ips-v6; do \
		echo "# $$u" >> snippets/cloudflare-allow.conf; \
		for ip in $$(curl -sSL "$$u"); do \
			echo "allow $$ip;" >> snippets/cloudflare-allow.conf; \
		done; \
	done

.PHONY: generate-cloudflare
