#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
  echo "usage: $0 command [args...]" >&2
  exit 2
fi

exec envsetter run -- "$@"
