#!/bin/sh

log() {
	local t="$(date '+%Y-%m-%dT%H:%M:%SZ' --utc)"
	local level=info
	echo "{\"time\":\"${t}\",\"level\":\"${level}\",\"message\":\"$@\"}"
}

error() {
	local t="$(date '+%Y-%m-%dT%H:%M:%SZ' --utc)"
	local level=error
	echo "{\"time\":\"${t}\",\"level\":\"${level}\",\"message\":\"$@\"}" 1>&2
}

fatal() {
	local t="$(date '+%Y-%m-%dT%H:%M:%SZ' --utc)"
	local level=fatal
	echo "{\"time\":\"${t}\",\"level\":\"${level}\",\"message\":\"$@\"}" 1>&2
}