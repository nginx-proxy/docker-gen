#!/bin/sh

set -eu

# If no arguments are passed, default to running docker-gen
if [ "$#" -eq 0 ]; then
	set -- docker-gen
else
	# Prepend docker-gen unless $1 resolves to an executable regular file
	_cmd=''
	if _cmd="$(command -v -- "$1" 2>/dev/null)" && [ -f "$_cmd" ] && [ -x "$_cmd" ]; then
		:
	else
		set -- docker-gen "$@"
	fi
	unset _cmd
fi

exec "$@"
