# fsync

[![License](https://img.shields.io/github/license/pjoc-team/go-action-template.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Stars](https://img.shields.io/github/stars/pjoc-team/go-action-template.svg)](https://github.com/pjoc-team/go-action-template/stargazers)
[![Builder](https://github.com/pjoc-team/go-action-template/workflows/Builder/badge.svg)](https://github.com/pjoc-team/go-action-template/actions)
[![Release](https://img.shields.io/github/v/tag/pjoc-team/pay-gateway)](https://github.com/pjoc-team/go-action-template/tags)
[![GoDoc](https://img.shields.io/badge/doc-go.dev-informational.svg)](https://pkg.go.dev/github.com/pjoc-team/go-action-template)
[![GoMod](https://img.shields.io/github/go-mod/go-version/pjoc-team/go-action-template.svg)](https://golang.org/)

[![Docker](https://img.shields.io/docker/v/pjoc/go-action-template.svg?label=docker)](https://hub.docker.com/r/pjoc/go-action-template/tags)
[![Docker](https://img.shields.io/docker/image-size/pjoc/go-action-template/latest.svg)](https://hub.docker.com/r/pjoc/go-action-template/tags)
[![Docker](https://img.shields.io/docker/pulls/pjoc/go-action-template.svg)](https://hub.docker.com/r/pjoc/go-action-template/tags)

A file sync server.

## Config

Cloud adaptor:
- TencentCloud: https://cloud.tencent.com/document/product/436/37421
- AliCloud: https://help.aliyun.com/document_detail/64919.html
- GoogleCloud: https://cloud.google.com/docs/compare/aws/storage#distributed_object_storage
- Azure: https://docs.microsoft.com/en-us/azure/storage/common/storage-account-create?tabs=azure-portal

### Parameters

| arg | env | description | default |
| --- | --- | --- | --- |
| data-path | DATA_PATH | Data path to upload and watch | ./data |
| secret-id | SECRET_ID | SecretID for s3 | |
| secret-key | SECRET_KEY | SecretKey for s3 | |
| bucket | BUCKET | Bucket name | backup-1251070767 |
| endpoint | ENDPOINT | Endpoint url for s3 | https://cos.ap-guangzhou.myqcloud.com |
| block-size | BLOCK_SIZE | upload buf | 1048576 |
| debug | DEBUG | is debug log? | true |
| init-upload | INIT_UPLOAD | need upload all data. | false |

## Docker

see [./docker/start.sh](docker/start.sh)

```bash
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

```

## Collaborators

<!-- readme: collaborators -start --> 
<table>
</table>
<!-- readme: collaborators -end -->

## Contributors

<!-- readme: contributors -start --> 
<table>
<tr>
    <td align="center">
        <a href="https://github.com/blademainer">
            <img src="https://avatars.githubusercontent.com/u/3396459?v=4" width="100;" alt="blademainer"/>
            <br />
            <sub><b>Blademainer</b></sub>
        </a>
    </td></tr>
</table>
<!-- readme: contributors -end -->