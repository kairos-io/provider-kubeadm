#!/bin/bash -x

LOG_FILE=/var/log/import.log
exec 1>$LOG_FILE 2>&1

# find all tar files recursively
for tarfile in $(find /opt/content -name "*.tar" -type f)
do
  # try to import the tar file into containerd up to ten times
  for i in {1..10}
  do
    ctr -n k8s.io image import $tarfile --all-platforms
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