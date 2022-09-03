ARG REDIS_VERSION=6.2.6-alpine
FROM redis:${REDIS_VERSION}
RUN apk add --update-cache --latest bind-tools && \
	mkdir -p /usr/local/share/redis
COPY scripts /usr/local/share/redis/scripts

COPY *.sh /usr/local/bin/
ENTRYPOINT ["entrypoint.sh"]