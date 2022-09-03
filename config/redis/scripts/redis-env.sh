#!/bin/sh

# Basic
export REDIS_PORT="${REDIS_PORT:-6379}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
export REDIS_DATA_DIR=/data

# config
export REDIS_CONFIG_DIR="${REDIS_CONFIG_DIR:-/etc/redis}"
export REDIS_CONFIG_FILE="${REDIS_CONFIG_DIR}/redis.conf"
export REDIS_DIR="${REDIS_DIR:-/data}"
export REDIS_DBFILENAME="${REDIS_DBFILENAME:-dump.rdb}"
export REDIS_APPENDONLY="${REDIS_APPENDONLY:-yes}"
export REDIS_APPENDFILENAME="${REDIS_APPEND_FILENAME:-appendonly.aof}"
export REDIS_CLUSTER_ENABLED="${REDIS_CLUSTER_ENABLED:-no}"
export REDIS_CLUSTER_NODE_TIMEOUT="${REDIS_CLUSTER_NODE_TIMEOUT:-5000}"

# Replication control
export REDIS_BASE_DOMAIN="${REDIS_BASE_DOMAIN:-}"
export REDIS_DEFAULT_MASTER="${REDIS_DEFAULT_MASTER:-false}"
export REDIS_DEFAULT_MASTER_DNS="${REDIS_DEFAULT_MASTER_DNS:-}"

# Sentinel
export REDIS_SENTINEL_PASSWORD="${REDIS_SENTINEL_PASSWORD:-}"
export REDIS_SENTINEL_MASTER_NAME="${REDIS_SENTINEL_MASTER_NAME:-}"
export REDIS_SENTINEL_PORT="${REDIS_SENTINEL_PORT:-26379}"
export REDIS_SENTINEL_HOST="${REDIS_SENTINEL_HOST:-sentinel}"

# Cluster Mode
export REDIS_CLUSTER_HOSTS="${REDIS_CLUSTER_HOSTS:-}"