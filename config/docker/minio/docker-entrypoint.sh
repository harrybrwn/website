#!/bin/bash
#

# If command starts with an option, prepend minio.
if [ "${1}" != "minio" ]; then
    if [ -n "${1}" ]; then
        set -- minio "$@"
    fi
fi

log() {
    echo "{\"message\": \"$@\", \"level\": \"info\"}"
}

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

json_log_line() {
    tr -d '\n' | sed 's/$/\n/g'
}

## Switch to user if applicable.
docker_switch_user "$@" &

POLICIES_DIR=/docker-entrypoint-init.d/policies

if [ -n "${HB_MC_ALIAS_NAME}" ]; then
    TARGET="${HB_MC_ALIAS_NAME}"
    # Create policies
    if [ -d "${POLICIES_DIR}" ]; then
        for file in $(ls -A "${POLICIES_DIR}"); do
        	name="$(echo ${file} | sed 's/\.json$//g')"
            f="${POLICIES_DIR}/${file}"
            log "creating policy ${name} from ${f}"
            mc --json admin policy add "${TARGET}" "${name}" "${f}" | json_log_line
        done
    fi
    # Create buckets
    if [ -n "${HB_DEFAULT_BUCKETS}" ]; then
        for bucket in $(echo "${HB_DEFAULT_BUCKETS}" | sed 's/,/ /g'); do
            log "creating bucket ${bucket}"
            mc --json mb "${TARGET}/${bucket}" | json_log_line
        done
    fi
    # Create users
    if [ -n "${HB_DEFAULT_USERS}" ]; then
        log 'error: user creation on startup is not yet supported'
    fi
fi

# Put minio server back in the forground
fg