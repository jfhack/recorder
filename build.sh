#!/bin/bash

VERSION=$(cat version | tr -d '[:space:]')

build_linux() {
  local arch=$1
  DIR="build/${arch}_${VERSION}"
  mkdir -p "$DIR"
  cp ./install.sh "$DIR/install.sh"
  CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${VERSION}" -o "${DIR}/recorder" ./cmd/recorder
  PARENT=$(dirname "$DIR")
  BASE=$(basename "$DIR")
  tar -C "$PARENT" -I 'gzip -9' -cf "$DIR.tar.gz" "$BASE"
  rm -rf "$DIR"
  echo "Built and compressed for $arch: $DIR.tar.gz"
}

types=("linux_amd64" "linux_armv7" "linux_arm64")
for type in "${types[@]}"; do
  echo "Building for $type..."
  if [[ "$type" == "linux_armv7" ]]; then
    GOARCH=arm GOARM=7 build_linux $type
    else
    GOARCH=${type#linux_} build_linux $type
  fi
done
