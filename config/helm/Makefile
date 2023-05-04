URL=https://helm.hrry.dev

help:
	@echo "make -C config/helm <target>"
	@echo
	@echo "Targets"
	@echo "  help"
	@echo "  build"
	@echo "  deploy"
	@echo "  clean"

build: dist/index.yaml

deploy: dist/index.yaml
	mc cp dist/index.yaml dist/*.tgz r2/helm
	helm repo remove hrry
	helm repo add hrry $(URL)

clean:
	$(RM) -r dist/ $(shell find . -type f -name '*.tgz')

dist: charts/mastodon
	sed -i 's/"common\.names\.fullname"/"mastodon\.fullname"/g;' charts/mastodon/templates/_helpers.tpl
	helm package --dependency-update --destination $@ charts/*

dist/index.yaml: dist
	helm repo index ./dist --url $(URL)

charts/mastodon:
	git clone --depth 1 git@github.com:mastodon/chart.git charts/mastodon

clone:
	@if [ ! -d charts/mastodon ]; then \
		git clone --depth 1 git@github.com:mastodon/chart.git charts/mastodon; \
	fi
	sed -i 's/"common\.names\.fullname"/"mastodon\.fullname"/g;' charts/mastodon/templates/_helpers.tpl


full-clean: clean
	$(RM) -r charts/mastodon

.PHONY: build deploy clean clone full-clean
