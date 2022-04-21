#!/bin/bash

declare -r CERTDB="sql:$HOME/.pki/nssdb"
declare -r LOCAL_CERT_NAME="harrybrwn-local-dev"

in_docker() {
	grep -q docker /proc/1/cgroup || [ -f /.dockerenv ]
}

has_certutil() {
	command -v certutil > /dev/null 2>&1
}