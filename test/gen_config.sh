#!/bin/bash
set -e
./binst init --source goreleaser --repo reviewdog/reviewdog -o=testdata/reviewdog.binstaller.yml --sha='7e05fa3e78ba7f2be4999ca2d35b00a3fd92a783'
./binst init --source goreleaser --repo actionutils/sigspy -o=testdata/sigspy.binstaller.yml --sha='3e1c6f32072cd4b8309d00bd31f498903f71c422'
