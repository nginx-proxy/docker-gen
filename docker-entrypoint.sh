#!/bin/sh
set -e

# if command starts with an option, prepend docker-gen
if [ "${1:0:1}" = '-' ]; then
	set -- docker-gen "$@"
fi

# Compute the DNS resolvers for use in the templates
export RESOLVERS=$(awk '$1 == "nameserver" {print $2}' ORS=' ' /etc/resolv.conf | sed 's/ *$//g')
if [ "x$RESOLVERS" = "x" ]; then
	echo "Warning: unable to determine DNS resolvers" >&2
fi

exec "$@"
