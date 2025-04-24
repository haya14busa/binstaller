#!/bin/bash
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"' EXIT HUP INT TERM

ls testdata/*.install.sh | rush -j5 -k "{} -b ${TMPDIR}"
