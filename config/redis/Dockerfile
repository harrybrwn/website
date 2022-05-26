ARG REDIS_VERSION=6.2.6-alpine
FROM redis:${REDIS_VERSION}
COPY config/redis/entrypoint.sh /usr/local/bin/entrypoint.sh
ENTRYPOINT ["entrypoint.sh"]