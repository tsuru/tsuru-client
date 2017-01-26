#!/bin/bash

NAME="tsuru"
VERSION=$1
if [ "$VERSION" == "" ]
then
    VERSION=$(cat tsuru/main.go | grep "version =" | cut -d '"' -f2)
fi
OSSES="darwin linux windows"
ARCHS="amd64 386"

mkdir dist || true

for os in $OSSES; do
    for arch in $ARCHS; do
        if [[ $os == "darwin" ]] && [[ $arch == "386" ]]; then
            continue
        fi
        echo "Building version $VERSION for $os $arch"
        dest="dist/${NAME}"
        zipname="dist/${NAME}-${VERSION}-${os}_${arch}"
        GOOS=$os GOARCH=$arch go build -o $dest ./tsuru
        if [[ $os == "windows" ]]; then
            mv $dest "${dest}.exe"
            zip "${zipname}.zip" "${dest}.exe"
            tar -zcpvf "${zipname}.tar.gz" "${dest}.exe"
            rm "${dest}.exe"
        else
            tar -zcpvf "${zipname}.tar.gz" $dest
            rm $dest
        fi
    done
done
