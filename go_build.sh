#!/usr/bin/env bash

export GO111MODULE=on

cmd_dir="./"
#set -x
bin_dir="./bin"
if [ -n "${BIN}" ]; then
    bin_dir="${BIN}"
fi
# Find main() func and build to bin
# For example, build source "cmd/app/main.go" to ./bin/app
grep -Er "func\s+main\(\s*\)" "${cmd_dir}" | awk -F ":" '{print $1}' | while read source; do
  # remove ${cmd_dir} prefix
  d=`cd ${source%/*} && pwd`
  dir_name="${d##*/}"
#  dir_name=`echo ${source%/*} | sed "s~${cmd_dir}~~"`
  bin="${bin_dir}/$dir_name"

  echo "build source: $d to bin: ${bin}"
  CGO_ENABLED=0 GOOS=linux go build -o ${bin} $d
done