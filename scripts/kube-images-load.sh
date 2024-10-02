#!/bin/bash

# Kubeadm Load Config Images
#
# This script will download all kubeadm config images for a specific kubeadm/k8s version using crane command.
#
# Usage:
#   $0 $kubeadm_version

set -ex

KUBE_VERSION=$1

ARCH=$(uname -m)
OS=$(uname)
ARCHIVE_NAME=go-containerregistry_"${OS}"_"${ARCH}".tar.gz
TEMP_DIR=/opt/kubeadm-temp
IMAGES_DIR=/opt/kube-images
IMAGE_FILE=images.list

# create temp dir
mkdir -p $TEMP_DIR && mkdir -p $IMAGES_DIR
cd $TEMP_DIR || exit

verify_downloader() {
  cmd="$(command -v "${1}")"
  if [ -z "${cmd}" ]; then
      return 1
  fi
  if [ ! -x "${cmd}" ]; then
      return 1
  fi

  DOWNLOADER=${cmd}
  return 0
}

download_crane() {
  verify_downloader curl || verify_downloader wget
  case ${DOWNLOADER} in
  *curl)
    curl -L -o "${ARCHIVE_NAME}" https://github.com/google/go-containerregistry/releases/download/v0.13.0/"${ARCHIVE_NAME}"
    ;;
  *wget)
    wget https://github.com/google/go-containerregistry/releases/download/v0.13.0/"${ARCHIVE_NAME}"
    ;;
  *)
    echo "curl or wget not found"
    exit 1
  esac
}

download_crane
tar -xvf "${ARCHIVE_NAME}" crane

# Put all kubeadm image into a file
kubeadm config images list --kubernetes-version "${KUBE_VERSION}" > $IMAGE_FILE

# create tar
while read -r image; do
  IFS="/" read -r -a im <<< $image
  image_name_with_version="${im[-1]}"

  IFS=":" read -r -a ima <<< $image_name_with_version
  name="${ima[0]}"
  version="${ima[1]}"

  ./crane pull "$image" ${IMAGES_DIR}/"${name}"-"${version}".tar
done < $IMAGE_FILE

rm -rf ${TEMP_DIR}