#!/bin/sh

disk="disk1"
names="cloudflare-address-list.rsc port-forwarding.rsc"

scripts=""
for name in $names; do
	scripts="$scripts scripts/router/$name"
done


init=""
for name in $names; do
	sname=$(echo $name | cut -d. -f1)
	echo "$sname"
	content="/system script remove $sname;"
	content="$content\n/system script add name=$sname source=[ /file get $disk/$name contents ];"
	if [ -n "$init" ]; then
		init="$init\n$content"
	else
		init="$content"
	fi
done

scp $scripts "admin@router.lan:$disk"
echo $init | ssh admin@router.lan
