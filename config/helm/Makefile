URL=https://helm.hrry.dev

deploy: dist/index.yaml
	mc cp dist/index.yaml dist/*.tgz r2/helm
	helm repo remove hrry
	helm repo add hrry $(URL)

clean:
	$(RM) -r dist/ $(shell find . -type f -name '*.tgz')

clone:
	@if [ ! -d charts/mastodon ]; then \
		git clone git@github.com:mastodon/chart.git charts/mastodon; \
	fi

dist: clone
	helm package --dependency-update --destination $@ charts/*

dist/index.yaml: dist
	helm repo index ./dist --url $(URL)


full-clean: clean
	$(RM) -r charts/mastodon

.PHONY: package deploy clean clone full-clean
