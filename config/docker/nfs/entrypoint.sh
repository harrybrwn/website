#!/bin/bash

set -e

CLIENT=false
mounts=()

while [ $# -gt 0 ]; do
	case $1 in
		--client)
			CLIENT=true
			shift 1
			;;
		*)
		  mounts+=("$1")
			shift 1
			;;
	esac
done

if ${CLIENT}; then
	NFS_PORT_2049_TCP_ADDR=nfs
	rpcbind

	targets=()
	for mnt in "${mounts[@]}"; do
		src=$(echo $mnt | awk -F':' '{ print $1 }')
		target=$(echo $mnt | awk -F':' '{ print $2 }')
		targets+=("$target")

		mkdir -p $target

		mount -t nfs -o proto=tcp,port=2049 ${NFS_PORT_2049_TCP_ADDR}:${src} ${target}
	done
	exec inotifywait -m "${targets[@]}"
else
	mkdir -p /mnt/nfs
	echo "/mnt/nfs *(rw,sync,no_subtree_check,insecure,no_root_squash,fsid=0)" >> /etc/exports
	exec runsvdir /etc/sv
fi
