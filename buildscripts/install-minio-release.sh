#!/bin/bash
#
# Builds an older MinIO server release from source and copies it to the given
# path. Used by upgrade tests now that dl.min.io and the minio/minio Docker Hub
# images are no longer available to the community.
#
# usage: install-minio-release.sh <release-tag> <dest>
#   e.g. install-minio-release.sh RELEASE.2024-03-26T22-10-45Z ./minio.RELEASE.2024-03-26T22-10-45Z

set -e

version="$1"
dest="$2"
gobin="$(go env GOPATH)/bin/minio-${version}"

if [ ! -x "${gobin}/minio" ]; then
	mkdir -p "${gobin}"
	# Releases whose go.mod has replace directives can't be built with
	# `go install pkg@version`; fall back to building a checkout of the tag.
	if ! GOBIN="${gobin}" GOFLAGS= CGO_ENABLED=0 go install "github.com/minio/minio@${version}"; then
		src="$(mktemp -d)"
		git clone --quiet --depth 1 --branch "${version}" https://github.com/minio/minio "${src}"
		(cd "${src}" && GOFLAGS=-mod=mod CGO_ENABLED=0 go build -o "${gobin}/minio" .)
		rm -rf "${src}"
	fi
fi

cp "${gobin}/minio" "${dest}"
chmod +x "${dest}"
