#!/bin/bash

IMAGE="abates314/ampserver"
CMDS="ampserver"
ARCHS="386 arm amd64 arm64"

build () {
  export GOARCH=$1
  CMD=$2
  mkdir -p dist/$GOARCH
  echo "Build: dist/$GOARCH/$CMD"
  GOOS=linux go build -o dist/$GOARCH/$CMD ./cmd/$CMD
}

for CMD in $CMDS ; do
  for ARCH in $ARCHS ; do
    build $ARCH $CMD
  done
done

echo "Building Docker images"
docker buildx build --push --platform linux/arm,linux/386,linux/amd64 -t $IMAGE .
