ARG MINIO_VERSION=RELEASE.2022-05-23T18-45-11Z.fips
FROM minio/minio:${MINIO_VERSION}
ARG MC_VERSION=RELEASE.2022-05-09T04-08-26Z.fips
ARG TARGETPLATFORM
ARG TARGETARCH=amd64
ARG BUILDPLATFORM
ARG BUILDARCH
RUN echo https://dl.min.io/client/mc/release/linux-${TARGETARCH}/archive/mc.${MC_VERSION}
RUN curl -fsSL -o /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-${TARGETARCH}/archive/mc.${MC_VERSION} && \
	chmod +x /usr/local/bin/mc

COPY docker/minio/docker-entrypoint.sh /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]
