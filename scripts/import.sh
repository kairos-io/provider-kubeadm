#!/bin/bash -x

CONTENT_PATH=$1

# find all tar files recursively
for tarfile in $(find $CONTENT_PATH -name "*.tar" -type f)
do
  # try to import the tar file into containerd up to ten times
  for i in {1..10}
  do
    if [ -S /run/spectro/containerd/containerd.sock ]; then
      /opt/bin/ctr -n k8s.io --address /run/spectro/containerd/containerd.sock image import "$tarfile" --all-platforms
    else
      /opt/bin/ctr -n k8s.io image import "$tarfile" --all-platforms
    fi
    if [ $? -eq 0 ]; then
      echo "Import successful: $tarfile (attempt $i)"
      break
    else
      if [ $i -eq 10 ]; then
        echo "Import failed: $tarfile (attempt $i)"
      fi
    fi
  done
done