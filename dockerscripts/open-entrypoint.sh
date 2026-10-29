#!/bin/sh
#
# Entrypoint da imagem MinIO Open (Dockerfile.open).
#
#   docker run minio-open                -> minio server $MINIO_VOLUMES $MINIO_OPTS
#   docker run minio-open server /x ...  -> minio server /x ...
#   docker run minio-open mc ls local    -> qualquer outro comando
#
# Na subida do servidor, roda o minio-bootstrap em segundo plano (buckets,
# acesso publico, usuario do Typebot e guia pos-deploy nos logs).
# Desative com MINIO_INIT_ENABLE=off.

set -e

if [ "$#" -eq 0 ] || { [ "$1" = "minio" ] && [ "$#" -eq 1 ]; }; then
	# shellcheck disable=SC2086
	set -- minio server ${MINIO_VOLUMES:-/data} ${MINIO_OPTS}
elif [ "${1#-}" != "$1" ] || [ "$1" = "server" ]; then
	set -- minio "$@"
fi

if [ "$1" = "minio" ] && [ "$2" = "server" ]; then
	if [ "${MINIO_INIT_ENABLE:-on}" != "off" ]; then
		/usr/bin/minio-bootstrap &
	fi
fi

exec "$@"
