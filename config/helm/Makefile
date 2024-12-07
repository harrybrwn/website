URL=https://helm.hrry.dev
TMP=.tmp

help:
	@echo "make -C config/helm <target>"
	@echo
	@echo "Targets"
	@echo "  help    show this help message"
	@echo "  build   build the helm charts locally"
	@echo "  test    run some basic tests"
	@echo "  deploy  deploy the helm charts"
	@echo "  clean   remove local files"

build: dist/index.yaml

test: build
	helm template minio-test ./dist/minio*.tgz -f values/minio-test.yaml

deploy: dist/index.yaml
	AWS_REGION=wnam mc cp dist/index.yaml dist/*.tgz r2/helm
	helm repo remove hrry || true
	helm repo add hrry $(URL)

clean:
	$(RM) -r dist/ $(shell find . -type f -name '*.tgz')

dist: charts/mastodon charts/minio charts/pds
	sed -i 's/"common\.names\.fullname"/"mastodon\.fullname"/g;' charts/mastodon/templates/_helpers.tpl
	helm package --dependency-update --destination $@ charts/minio charts/wordpress charts/pds

dist/index.yaml: dist
	helm repo index ./dist --url $(URL)

charts/mastodon:
	git clone --depth 1 git@github.com:mastodon/chart.git charts/mastodon

.PHONY: charts/minio
charts/minio:
	@#git clone --depth 1 git@github.com:minio/minio.git charts/minio
	@if [ ! -d $(TMP)/minio ]; then \
		git clone --branch custom-command-extra-volumes git@github.com:harrybrwn/minio.git $(TMP)/minio; \
	else \
		rm -rf charts/minio/; \
	fi
	cp -r $(TMP)/minio/helm/minio charts/minio

.PHONY: charts/pds
charts/pds:
	@if [ ! -L charts/pds ]; then ln -s "$$HOME/dev/bluesky/pds/chart" charts/pds; fi

clone:
	@if [ ! -d charts/mastodon ]; then \
		git clone --depth 1 git@github.com:mastodon/chart.git charts/mastodon; \
	fi
	sed -i 's/"common\.names\.fullname"/"mastodon\.fullname"/g;' charts/mastodon/templates/_helpers.tpl

full-clean: clean
	$(RM) -r charts/mastodon

.PHONY: build deploy clean clone full-clean
