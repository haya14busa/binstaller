#!/bin/bash
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"' EXIT HUP INT TERM
yq -r '.supported_platforms[] | "\(.os) \(.arch)"' testdata/reviewdog.binstaller.yml |
  rush -d ' ' -j5 -k \
    "mkdir -p ${TMPDIR}/{1}-{2} && BINSTALLER_OS={1} BINSTALLER_ARCH={2} testdata/reviewdog.install.sh -b ${TMPDIR}/{1}-{2}"
