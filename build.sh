#!/bin/bash
REPO_NAME=appf

set -eo pipefail
image_name=`cat $1/Dockerfile|grep FROM | cut -d' ' -f2`

if [ ! -f "$1/.manifest_$image_name.json" ] ;then
    echo "no manifest, getting new manifest for $image_name"
    docker manifest inspect $image_name > "$1/.manifest_$image_name.json"
else
    echo "existing manifest found for $image_name"
fi
architectures=`cat $1/.manifest_$image_name.json | jq -r '.manifests[] | .platform.architecture'`
echo "building for architectures: $architectures"

function join_by { local IFS="$1"; shift; echo "$*"; }

# docker run --rm --privileged multiarch/qemu-user-static:register --reset
name=$(basename $1)
cur=`pwd`
cd "$1"
platforms=()
for arch in $architectures
do
    echo "Compiling for $arch"
    CGO_ENABLED=0 GOARCH=$arch GOARM=6 go build -a -o "$name"  &
    PID=$!
    i=1
    sp="/-\|"
    echo -n ' '
    while [ -d /proc/$PID ]
    do
      sleep 0.1
      printf "\b${sp:i++%${#sp}:1}"
    done
    echo ""
    echo "Compiled successfully for $arch"
    echo "Building docker image for $arch..."
    image_repo=`echo -e $image_name|cut -d':' -f1` 
    sha256hash=`cat .manifest_$image_name.json | \
        jq -r ".manifests[] | select(.platform.architecture == \"$arch\") | .digest"`
    image="$image_repo@$sha256hash"
    # cat Dockerfile | sed "s/^FROM.*\$/FROM $image/" > Dockerfile.$arch
    cat Dockerfile | sed "s/^FROM.*\$/FROM $image/" | QEMU_ARCH=$arch docker build -t $REPO_NAME/$name:$arch -f - . && \
        docker push $REPO_NAME/$name:$arch
    platforms+=("linux/$arch")
    rm "$name"
done
cd $cur
platforms_str=`join_by , "${platforms[@]}"`

manifest-tool push from-args \
 --platforms $platforms_str \
 --template $REPO_NAME/$name:ARCH \
 --target $REPO_NAME/$name:latest
docker manifest inspect $REPO_NAME/$name