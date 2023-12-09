#!/bin/sh

set -eu

# run container's CMD if it is an executable in PATH
command -v -- "$1" >/dev/null 2>&1 || set -- docker-gen "$@"

exec "$@"
