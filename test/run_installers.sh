#!/bin/bash
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"' EXIT HUP INT TERM

ls testdata/*.install.sh | xargs -I {} -P5 sh -c "{} -b ${TMPDIR}"
