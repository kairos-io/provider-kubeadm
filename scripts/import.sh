#!/bin/sh

CONTENT_PATH=$1

if [ -S /run/spectro/containerd/containerd.sock ]; then
  CTR_SOCKET=/run/spectro/containerd/containerd.sock
else
  CTR_SOCKET=/run/containerd/containerd.sock
fi

import_image() {
  tarfile=$1
  i=1
  while [ $i -le 10 ]; do
    output=$(/opt/bin/ctr -n k8s.io --address $CTR_SOCKET image import "$tarfile" --all-platforms 2>&1)
    exit_code=$?

    if [ $exit_code -eq 0 ]; then
      echo "Import successful: $tarfile (attempt $i)"
      break
    elif echo "$output" | grep -q "ctr: image might be filtered out"; then
      echo "Import skipped (filtered out): $tarfile (attempt $i)"
      break
    else
      if [ $i -eq 10 ]; then
        echo "Import failed: $tarfile (attempt $i)"
      fi
    fi
    i=$((i + 1))
  done
}

# find all tar files recursively
find -L "$CONTENT_PATH" -name "*.tar" -type f | while read -r tarfile; do
  echo "Importing: $tarfile"
  import_image "$tarfile"
done
