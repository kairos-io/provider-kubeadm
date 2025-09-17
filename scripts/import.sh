#!/bin/bash

CONTENT_PATH=$1
LOG_FILE=$2

exec > >(tee -ia "$LOG_FILE")
exec 2> >(tee -ia "$LOG_FILE" >&2)
exec 19>>$LOG_FILE

echo "--------------------------------"
echo "Importing images from $CONTENT_PATH at $(date)"

if [ -S /run/spectro/containerd/containerd.sock ]; then
  CTR_SOCKET=/run/spectro/containerd/containerd.sock
else
  CTR_SOCKET=/run/containerd/containerd.sock
fi

import_image() {
  local tarfile=$1
  local i=1

  echo "Importing: $tarfile"
  for i in {1..10}; do
    output=$(/opt/bin/ctr -n k8s.io --address $CTR_SOCKET image import "$tarfile" --all-platforms 2>&1)
    exit_code=$?

    if [ $exit_code -eq 0 ]; then
      echo "Import successful: $tarfile (attempt $i)"
      break
    elif echo "$output" | grep -q "ctr: image might be filtered out"; then
      echo "Import skipped (filtered out): $tarfile (attempt $i)"
      break
    else
      echo "Import failed: $tarfile exit code: $exit_code (attempt $i)"
      echo "Output: $output"
    fi
    sleep 1
  done
}

# find all tar files recursively
find -L "$CONTENT_PATH" -name "*.tar" -type f | while read -r tarfile; do
  import_image "$tarfile"
done
