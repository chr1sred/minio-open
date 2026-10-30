#!/bin/bash
#
# Builds the MinIO Client (mc) from source and copies it to the given path.
# dl.min.io no longer serves community binaries, so test scripts use this
# instead of downloading mc.
#
# usage: install-mc.sh [dest] [version]
#   dest     default ./mc
#   version  module version or release tag, e.g. RELEASE.2021-03-12T03-36-59Z
#            (default: $MC_VERSION or the pinned version below)

set -e

dest="${1:-./mc}"
version="${2:-${MC_VERSION:-v0.0.0-20251106162529-77f82e18b540}}"
gobin="$(go env GOPATH)/bin/mc-${version}"

if [ ! -x "${gobin}/mc" ]; then
	GOBIN="${gobin}" GOFLAGS= go install "github.com/minio/mc@${version}"
fi

cp "${gobin}/mc" "${dest}"
chmod +x "${dest}"
