#!/bin/bash
echo "## Extracting app name and version from source"
APP_NAME=$(grep -E 'APP\s+=' server.go| awk '{ print $3 }'  | tr -d '"')
APP_VERSION=$(grep -E 'VERSION\s+=' server.go| awk '{ print $3 }'  | tr -d '"')
echo "## Found APP: ${APP_NAME}, VERSION: ${APP_VERSION}  in source file server.go"
export APP_VERSION APP_NAME
