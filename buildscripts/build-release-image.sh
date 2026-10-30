#!/bin/bash
#
# Builds a local Docker image for an older MinIO release from source, for
# tests that used to pull it from Docker Hub/quay.io (no longer published for
# the community).
#
# usage: build-release-image.sh <release-tag> <image:tag>
#   e.g. build-release-image.sh RELEASE.2019-12-19T22-52-26Z minio/minio:RELEASE.2019-12-19T22-52-26Z

set -e

release="$1"
image="$2"
root="$(cd "$(dirname "$0")/.." && pwd)"
ctx="$(mktemp -d)"

bash "${root}/buildscripts/install-minio-release.sh" "${release}" "${ctx}/minio"
printf 'FROM alpine:3.22\nCOPY minio /usr/bin/minio\nENTRYPOINT ["/usr/bin/minio"]\n' >"${ctx}/Dockerfile"
docker build -q -t "${image}" "${ctx}"
rm -rf "${ctx}"
