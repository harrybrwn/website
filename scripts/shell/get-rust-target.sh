#!/bin/sh

set -eu

if [ -z "${1:-}" ]; then
	echo "Error: no target platform given" 1>&2
	exit 1
fi

target="${1}"

hardfloat=false
arm_ext=""
gnu=false
musl=true
while [ $# -gt 0 ]; do
	case "$1" in
		--musl)
			musl=true
			gnu=false
			;;
		--gnu)
			gnu=true
			musl=false
			;;
		--hardfloat)
			hardfloat=true
			;;
		--arm-ext)
			arm_ext="$2"
			shift
			;;
		*)
			;;
	esac
	shift
done

case "${target}" in
	linux/amd64)
		t="x86_64-unknown-linux"
		if ${musl}; then
			t="${t}-musl"
		elif ${gnu}; then
			t="${t}-gnu"
		fi
		echo "${t}"
		;;
	linux/arm64)
	 	t="aarch64-unknown-linux"
	 	if ${musl}; then
			t="${t}-musl"
		elif ${gnu}; then
			t="${t}-gnu"
		fi
		echo "${t}"
		;;
	linux/arm/v7)
		t="armv7"
		if [ -n "${arm_ext}" ]; then
			t="${t}${arm_ext}"
		fi
		t="${t}-unknown-linux"
	 	if ${musl}; then
			t="${t}-musleabi"
		elif ${gnu}; then
			t="${t}-gnueabi"
		fi
		if ${hardfloat}; then
			t="${t}hf"
		fi
		echo "${t}"
		;;
	linux/arm/v6)
		t="arm-unknown-linux"
	 	if ${musl}; then
			t="${t}-musleabi"
		elif ${gnu}; then
			t="${t}-gnueabi"
		fi
		if ${hardfloat}; then
			t="${t}hf"
		fi
		echo "${t}"
		;;
	*)
		echo "Error: unknown target platform \"${1}\"" 1>&2
		exit 1
		;;
esac
