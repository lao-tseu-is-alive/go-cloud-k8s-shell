#!/bin/bash
CONTAINER_REGISTRY="ghcr.io/"
CONTAINER_REGISTRY_USER="lao-tseu-is-alive"
CONTAINER_REGISTRY_ID="${CONTAINER_REGISTRY}${CONTAINER_REGISTRY_USER}"
# checks whether APP_NAME has length equal to zero:
if [[ -z "${APP_NAME}" ]]
then
	echo "## ENV variable APP_NAME not found"
      	FILE=getAppInfo.sh
	if test -f "$FILE"; then
		echo "## Sourcing $FILE"
		# shellcheck disable=SC1090
		source $FILE
	elif test -f "./scripts/${FILE}"; then
		echo "## Sourcing ./scripts/$FILE"
  		# shellcheck disable=SC1090
  	source ./scripts/$FILE
	else
	  echo "## 💥💥 ERROR: getAppInfo.sh was not found"
		exit 1
	fi
else
	echo "## ENV variable APP_NAME is defined to : ${APP_NAME} . So we will use this one !"
fi
IMAGE_FILTER="${CONTAINER_REGISTRY_ID}/${APP_NAME}"
echo "## will scan for fixed vulnerabilities image : ${IMAGE_FILTER}:v${APP_VERSION}"
trivy image --ignore-unfixed "${IMAGE_FILTER}:v${APP_VERSION}"
