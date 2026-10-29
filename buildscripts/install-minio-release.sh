#!/bin/bash
#
# Builds an older MinIO server release from source and copies it to the given
# path. Used by upgrade tests now that dl.min.io no longer serves community
# binaries.
#
# usage: install-minio-release.sh <release-tag> <dest>
#   e.g. install-minio-release.sh RELEASE.2024-03-26T22-10-45Z ./minio.RELEASE.2024-03-26T22-10-45Z

set -e

version="$1"
dest="$2"
gobin="$(go env GOPATH)/bin/minio-${version}"

if [ ! -x "${gobin}/minio" ]; then
	GOBIN="${gobin}" GOFLAGS= go install "github.com/minio/minio@${version}"
fi

cp "${gobin}/minio" "${dest}"
chmod +x "${dest}"
