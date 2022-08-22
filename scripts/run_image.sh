#!/bin/bash
echo "will extract app name and version from source"
VERSION=`grep -E 'VERSION\s+=' server.go| awk '{ print $3 }'  | tr -d '"'`
APPNAME=`grep -E 'APP\s+=' server.go| awk '{ print $3 }'  | tr -d '"'`
echo "APP: ${APPNAME}, version: ${VERSION} detected in file server.go"
echo "listing relevant images in k8s namespace"
nerdctl -n k8s.io images | grep ${APPNAME}
nerdctl -n k8s.io run -it  -p 127.0.0.1:8080:8080 --rm $APPNAME

