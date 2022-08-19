# NFS Server in docker
# See References
#
# - https://github.com/cpuguy83/docker-nfs-server
# - https://github.com/cpuguy83/docker-nfs-client
# - https://github.com/sjiveson/nfs-server-alpine
# - https://container42.com/2014/03/29/docker-quicktip-4-remote-volumes/
# - https://hub.docker.com/r/erichough/nfs-server/
# - https://github.com/kubernetes/examples/blob/master/staging/volumes/nfs
#
ARG UBUNTU_VERSION=latest
FROM ubuntu:${UBUNTU_VERSION} as nfs-server
ENV DEBIAN_FRONTEND noninteractive
RUN apt update -q && \
	apt install -yq nfs-kernel-server runit inotify-tools && \
	mkdir -p /exports /etc/sv/nfs

RUN echo "nfs             2049/tcp" >> /etc/services && \
	echo "nfs             111/udp" >> /etc/services

COPY init /etc/sv/nfs/run
COPY stop /etc/sv/nfs/finish
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

VOLUME /exports

EXPOSE 111/udp 2049/tcp

ENTRYPOINT ["entrypoint.sh"]