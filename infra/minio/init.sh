#!/bin/sh
set -eu

echo "waiting for MinIO..."
i=0
until mc alias set local http://minio:9000 "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}" >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge 15 ]; then
    echo "failed to connect to MinIO" >&2
    exit 1
  fi
  sleep 1
done

mc mb --ignore-existing local/product-images
mc mb --ignore-existing local/exports

echo "buckets ready:"
mc ls local
