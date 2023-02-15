#!/bin/bash
FILE=getAppInfo.sh
if test -f "$FILE"; then
  echo "## will execute $FILE"
	# shellcheck disable=SC1090
	source $FILE
elif test -f "./scripts/${FILE}"; then
  echo "## will execute ./scripts/$FILE"
  # shellcheck disable=SC1090
  source ./scripts/$FILE
else
  echo "-- ERROR getAppInfo.sh was not found"
  exit 1
fi
echo "## APP: ${APP_NAME}, version: ${APP_VERSION} detected in file server.go"
if output=$(git status --porcelain) && [ -z "$output" ]; then
  if [ $(git tag -l "v$APP_VERSION") ]; then
    echo "## 💥💥 ERROR: \"${APP_NAME} tag ${APP_VERSION} \" already exist !"
  else
    echo "## ✓🚀 OK: ${APP_NAME} tag ${APP_VERSION}  was not found ! So let's add this tag..."
    git tag "v$APP_VERSION" -m "v$APP_VERSION bump"
    git push origin --tags
  fi
else
  echo "## 💥💥 ERROR: \"${APP_NAME} source tree is DIRTY YOU MUST commit your code before doing a tag ${APP_VERSION} !\""
  git status
fi



