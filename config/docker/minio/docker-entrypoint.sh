#!/bin/bash
#

# If command starts with an option, prepend minio.
if [ "${1}" != "minio" ]; then
    if [ -n "${1}" ]; then
        set -- minio "$@"
    fi
fi

# su-exec to requested user, if service cannot run exec will fail.
docker_switch_user() {
    if [ -n "${MINIO_USERNAME}" ] && [ -n "${MINIO_GROUPNAME}" ]; then
        if [ -n "${MINIO_UID}" ] && [ -n "${MINIO_GID}" ]; then
            groupadd -g "$MINIO_GID" "$MINIO_GROUPNAME" && \
                useradd -u "$MINIO_UID" -g "$MINIO_GROUPNAME" "$MINIO_USERNAME"
        else
            groupadd "$MINIO_GROUPNAME" && \
                useradd -g "$MINIO_GROUPNAME" "$MINIO_USERNAME"
        fi
        exec setpriv --reuid="${MINIO_USERNAME}" \
             --regid="${MINIO_GROUPNAME}" --keep-groups "$@"
    else
        exec "$@"
    fi
}

if [ -n "${HB_MC_ALIAS_NAME}" -a -n "${HB_MC_ALIAS_URL}" ]; then
    mkdir -p /root/.mc
    cat > /root/.mc/config.json <<-EOF
{
    "version": "10",
    "aliases": {
        "${HB_MC_ALIAS_NAME}": {
            "url": "${HB_MC_ALIAS_URL}",
            "accessKey": "${MINIO_ROOT_USER}",
            "secretKey": "${MINIO_ROOT_PASSWORD}",
            "api": "S3v4",
            "path": "auto"
        }
    }
}
EOF

fi

# Enable job control for initialization operations
set -m

## Switch to user if applicable.
docker_switch_user "$@" &

if [ -n "${HB_MC_ALIAS_NAME}" -a -n "${HB_DEFAULT_BUCKET}" ]; then
    mc mb "${HB_MC_ALIAS_NAME}/${HB_DEFAULT_BUCKET}" || true
fi

fg