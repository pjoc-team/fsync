#!/usr/bin/env bash

docker run --name="fsync-init" --rm -d \
        -v `pwd`/data:/data \
       	-e SECRET_ID="[YOUR_SECRET_ID]" \
       	-e SECRET_KEY="[YOUR_SECRET_KEY]" \
	      -e DATA_PATH="/data" \
        -e ENDPOINT="https://cos.ap-guangzhou.myqcloud.com" \
        -e BUCKET="backup-1251070767" \
        -e BLOCK_SIZE="1048576" \
        -e DEBUG="true" \
        -e INIT_UPLOAD="true" \
	pjoc/fsync:master

docker run --name="fsync-watcher" -d \
        -v `pwd`/data:/data \
       	-e SECRET_ID="[YOUR_SECRET_ID]" \
       	-e SECRET_KEY="[YOUR_SECRET_KEY]" \
	      -e DATA_PATH="/data" \
        -e ENDPOINT="https://cos.ap-guangzhou.myqcloud.com" \
        -e BUCKET="backup-1251070767" \
        -e BLOCK_SIZE="1048576" \
        -e DEBUG="true" \
        -e INIT_UPLOAD="false" \
	pjoc/fsync:master
