#!/bin/sh

set -eu

ip="$1"
if [ -z "$ip" ]; then
  echo "Error: must give new ip address"
  exit 1
fi

if ! echo "$1" | grep -E '(([0-9]{1,3}\.?){4}|localhost)' > /dev/null; then
  echo "input is not an ip address"
  exit 1
fi

sed -Ei "s/^(([0-9]{1,3}\.?){4}|localhost)[ ]+(.*?\.local)/${ip} \3/" /etc/hosts
cat /etc/hosts
