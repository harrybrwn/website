ARG ALPINE_VERSION=3.14
FROM alpine:${ALPINE_VERSION} as nomad-installer

ARG NOMAD_VERSION=1.3.5
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETVARIANT
ARG TARGETARCH
ARG BUILDPLATFORM
ARG BUILDARCH

RUN apk --update add --latest \
	ca-certificates \
	gnupg \
	gcompat

ADD https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_${TARGETOS}_${TARGETARCH}.zip \
    nomad_${NOMAD_VERSION}_${TARGETOS}_${TARGETARCH}.zip
ADD https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_SHA256SUMS \
    nomad_${NOMAD_VERSION}_SHA256SUMS
ADD https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_SHA256SUMS.sig \
    nomad_${NOMAD_VERSION}_SHA256SUMS.sig

RUN GNUPGHOME="$(mktemp -d)" && \
	export GNUPGHOME && \
	gpg --keyserver pgp.mit.edu --keyserver keys.openpgp.org --keyserver keyserver.ubuntu.com --recv-keys "C874 011F 0AB4 0511 0D02 1055 3436 5D94 72D7 468F" && \
	gpg --batch --verify nomad_${NOMAD_VERSION}_SHA256SUMS.sig nomad_${NOMAD_VERSION}_SHA256SUMS && \
	grep nomad_${NOMAD_VERSION}_${TARGETOS}_${TARGETARCH}.zip nomad_${NOMAD_VERSION}_SHA256SUMS | sha256sum -c && \
	unzip -d /bin nomad_${NOMAD_VERSION}_${TARGETOS}_${TARGETARCH}.zip && \
	chmod +x /bin/nomad

FROM alpine:${ALPINE_VERSION} as nomad

# Install glibc
ARG GLIBC_VERSION=2.33-r0
ADD https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub /etc/apk/keys/sgerrand.rsa.pub
ADD https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VERSION}/glibc-${GLIBC_VERSION}.apk \
    glibc.apk
RUN apk add --no-cache glibc.apk && rm glibc.apk

COPY --from=nomad-installer /bin/nomad /usr/bin/nomad

EXPOSE 4646 4647 4648 4648/udp
ENTRYPOINT [ "/usr/bin/nomad" ]