#!/bin/sh
set -eu

output=${1:-.env}
umask 077
exec envsetter pull --output "$output"
