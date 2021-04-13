#!/usr/bin/env bash

cur_script_dir="$(cd $(dirname $0) && pwd)"
WORK_HOME="${cur_script_dir}"

export REPOSITORY=`cat go.mod | grep -E "^module\s[0-9a-zA-Z\./_\-]+" | awk '{print $2}'`
export NAME=`basename $REPOSITORY`
export APP=`basename $REPOSITORY`
