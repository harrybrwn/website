#!/bin/sh
# vim:sw=4:ts=4:et

set -e

INDEX="${REGISTRY_UI_ROOT}/index.html"
if [ -n "${REGISTRY_URL}" -a -d "${REGISTRY_UI_ROOT}" -a -f "${INDEX}" ]; then
    sed -i "s~\${REGISTRY_URL}~${REGISTRY_URL}~" "${INDEX}"
    sed -i "s~\${REGISTRY_TITLE}~${REGISTRY_TITLE}~" "${INDEX}"
    sed -i "s~\${PULL_URL}~${PULL_URL}~" "${INDEX}"
    sed -i "s~\${SINGLE_REGISTRY}~${SINGLE_REGISTRY}~" "${INDEX}"
    sed -i "s~\${CATALOG_ELEMENTS_LIMIT}~${CATALOG_ELEMENTS_LIMIT}~" "${INDEX}"
    sed -i "s~\${SHOW_CONTENT_DIGEST}~${SHOW_CONTENT_DIGEST}~" "${INDEX}"
    sed -i "s~\${DEFAULT_REGISTRIES}~${DEFAULT_REGISTRIES}~" "${INDEX}"
    sed -i "s~\${READ_ONLY_REGISTRIES}~${READ_ONLY_REGISTRIES}~" "${INDEX}"
    sed -i "s~\${SHOW_CATALOG_NB_TAGS}~${SHOW_CATALOG_NB_TAGS}~" "${INDEX}"
    sed -i "s~\${HISTORY_CUSTOM_LABELS}~${HISTORY_CUSTOM_LABELS}~" "${INDEX}"
    if [ -z "${DELETE_IMAGES}" ] || [ "${DELETE_IMAGES}" = false ] ; then
        sed -i "s/\${DELETE_IMAGES}/false/" "${INDEX}"
    else
        sed -i "s/\${DELETE_IMAGES}/true/" "${INDEX}"
    fi
fi

if [ -z "${NGINX_ENTRYPOINT_QUIET_LOGS:-}" ]; then
    exec 3>&1
else
    exec 3>/dev/null
fi

if [ "$1" = "nginx" -o "$1" = "nginx-debug" ]; then
    if /usr/bin/find "/docker-entrypoint.d/" -mindepth 1 -maxdepth 1 -type f -print -quit 2>/dev/null | read v; then
        echo >&3 "$0: /docker-entrypoint.d/ is not empty, will attempt to perform configuration"

        echo >&3 "$0: Looking for shell scripts in /docker-entrypoint.d/"
        find "/docker-entrypoint.d/" -follow -type f -print | sort -V | while read -r f; do
            case "$f" in
                *.sh)
                    if [ -x "$f" ]; then
                        echo >&3 "$0: Launching $f";
                        "$f"
                    else
                        # warn on shell scripts without exec bit
                        echo >&3 "$0: Ignoring $f, not executable";
                    fi
                    ;;
                *) echo >&3 "$0: Ignoring $f";;
            esac
        done

        echo >&3 "$0: Configuration complete; ready for start up"
    else
        echo >&3 "$0: No files found in /docker-entrypoint.d/, skipping configuration"
    fi
fi

echo resolver $(awk 'BEGIN{ORS=" "} $1=="nameserver" {print $2}' /etc/resolv.conf) ";" >> /etc/nginx/conf.d/00-resolver.conf

exec "$@"
