#!/bin/sh

log() {
	t="$(date '+%Y-%m-%dT%H:%M:%SZ' --utc)"
	_log_level=info
	echo "{\"time\":\"${t}\",\"level\":\"${_log_level}\",\"message\":\"$*\"}"
  unset _log_level
  unset t
}

error() {
	t="$(date '+%Y-%m-%dT%H:%M:%SZ' --utc)"
	_log_level=error
	echo "{\"time\":\"${t}\",\"level\":\"${_log_level}\",\"message\":\"$*\"}" 1>&2
  unset _log_level
  unset t
}

fatal() {
	t="$(date '+%Y-%m-%dT%H:%M:%SZ' --utc)"
	_log_level=fatal
	echo "{\"time\":\"${t}\",\"level\":\"${_log_level}\",\"message\":\"$*\"}" 1>&2
  unset _log_level
  unset t
}
