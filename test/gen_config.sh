#!/bin/bash
set -e
./binst init --source goreleaser --repo reviewdog/reviewdog -o=testdata/reviewdog.binstaller.yml --sha='7e05fa3e78ba7f2be4999ca2d35b00a3fd92a783'
./binst init --source goreleaser --repo actionutils/sigspy -o=testdata/sigspy.binstaller.yml --sha='3e1c6f32072cd4b8309d00bd31f498903f71c422'
./binst init --source aqua --repo zyedidia/micro --output=testdata/micro.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
./binst init --source aqua --repo houseabsolute/ubi --output=testdata/ubi.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test rosetta2
./binst init --source aqua --repo ducaale/xh --output=testdata/xh.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test empty extension (extension hard coded in template)
./binst init --source aqua --repo Lallassu/gorss --output=testdata/gorss.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'

