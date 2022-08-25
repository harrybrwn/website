ARG REDIS_VERSION=6.2.6-alpine
FROM redis:${REDIS_VERSION}
RUN mkdir -p /usr/local/share/redis
COPY config/redis/scripts /usr/local/share/redis/scripts

COPY config/redis/entrypoint.sh /usr/local/bin/
COPY config/redis/node-entrypoint.sh /usr/local/bin/
COPY config/redis/sentinel-entrypoint.sh /usr/local/bin/
ENTRYPOINT ["entrypoint.sh"]