#!/bin/bash
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"' EXIT HUP INT TERM
yq '.supported_platforms[] | "BINSTALLER_OS=\(.os) BINSTALLER_ARCH=\(.arch)"' < testdata/reviewdog.binstaller.yml \
  | xargs -I {} -P5 sh -c "{} testdata/reviewdog.install.sh -b ${TMPDIR}"

