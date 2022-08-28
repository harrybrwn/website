docker_compose_images() {
  docker compose                                  \
    --file docker-compose.yml                     \
    --file config/docker-compose.logging.yml      \
    --file config/docker-compose.tools.yml config \
    | grep -E 'image:.*' \
    | awk '{ print $2 }' \
    | sort \
    | uniq
}