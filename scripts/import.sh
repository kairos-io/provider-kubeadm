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

# containerd >=2.1 defaults ctr image import to the Transfer Service (--local=false).
# The Transfer Service does not properly register images with the CRI plugin's
# reference index, so kubelet's ImageStatus call cannot find the image despite it
# being present in the content store.  This causes imagePullPolicy:IfNotPresent to
# trigger a registry pull, which fails on airgapped hosts.
# Adding --local forces the legacy client.Import path that calls ImageService().Create(),
# populating the CRI index correctly.
# See: https://github.com/containerd/containerd/issues/9039
CTR_LOCAL_FLAG=""
if ctr_version=$(/opt/bin/ctr --version 2>/dev/null); then
  # Extract version number (e.g. "containerd github.com/containerd/containerd/v2 v2.1.4 ..." → "2.1.4")
  ver=$(echo "$ctr_version" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 | sed 's/^v//')
  major=$(echo "$ver" | cut -d. -f1)
  minor=$(echo "$ver" | cut -d. -f2)
  if [ "${major:-0}" -ge 2 ] && [ "${minor:-0}" -ge 1 ]; then
    CTR_LOCAL_FLAG="--local"
    echo "containerd $ver detected (>=2.1), using --local flag for import"
  fi
fi

import_image() {
  local tarfile=$1
  local i=1

  echo "Importing: $tarfile"
  for i in {1..10}; do
    output=$(/opt/bin/ctr -n k8s.io --address $CTR_SOCKET image import "$tarfile" --all-platforms $CTR_LOCAL_FLAG 2>&1)
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
