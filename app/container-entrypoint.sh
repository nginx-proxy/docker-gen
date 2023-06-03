#!/bin/sh

set -eu

bin='docker-gen'

# run command if it is not starting with a "-" and is an executable in PATH
if [ "${#}" -le 0 ] || \
   [ "${1#-}" != "${1}" ] || \
   [ -d "${1}" ] || \
   ! command -v "${1}" > '/dev/null' 2>&1; then
	entrypoint='true'
fi

exec ${entrypoint:+${bin:?}} "${@}"

exit 0
