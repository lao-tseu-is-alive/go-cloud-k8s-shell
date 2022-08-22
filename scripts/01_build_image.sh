#!/bin/bash
echo "## will test for ENV variable APP_NAME"
# checks whether APP_NAME has length equal to zero:
if [[ -z "${APP_NAME}" ]]
then
	echo "## ENV variable APP_NAME not found"
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
else
	echo "## ENV variable APP_NAME found : ${APP_NAME}"
fi
# for k8s namespace
DOCKER_BIN="nerdctl -n k8s.io"
DOCKER_BIN="nerdctl"
if $DOCKER_BIN --version;
then
  echo "## will use \"${DOCKER_BIN}\" to build the container image on linux "
  CONTAINER_REGISTRY_ID=laotseu
  echo "## APP: ${APP_NAME}, version: ${APP_VERSION} detected in file server.go"
  IMAGE_FILTER="${CONTAINER_REGISTRY_ID}/${APP_NAME}"
  echo "## Checking if image:tag was already build in k8s namespace ${IMAGE_FILTER} tag:${APP_VERSION}"
  JSON_APP=$(${DOCKER_BIN} images --format '{{json .}}' | jq ".| select(.Repository | contains(\"${IMAGE_FILTER}\")) |select(.Tag | contains(\"${APP_VERSION}\"))")
  APP_ID=$(echo "${JSON_APP}" | jq '.|.ID')
  # checks whether APP_ID has length equal to zero --> meaning this image:version is not present and was probably not already build
  if [[ -z "${APP_ID}" ]]
  then
    echo "## Cool 🚀✓🚀 OK: ${IMAGE_FILTER}:${APP_VERSION} image was not found ! So let's try to build it..."
    TMP_Docker_Dir=$(mktemp -d)
    cp Dockerfile* "$TMP_Docker_Dir"
    cd "$TMP_Docker_Dir" || exit
    if trivy config --exit-code 1 --severity MEDIUM,HIGH,CRITICAL . ;
    #if [ $? -eq 0 ]
    then
      echo "## Cool 🚀✓🚀 OK: no vulnerabilities found in your Dockerfile will change directory :$OLDPWD"
      cd "$OLDPWD" || exit
      rm -rf "$TMP_Docker_Dir" # cleanup
      echo "will parse the multi-stage Dockerfile in the current directory and build the final image"
      if ${DOCKER_BIN} build -t ${CONTAINER_REGISTRY_ID}/"${APP_NAME}" . ;
      then
        echo "will tag this image with version ${APP_VERSION}"
        ${DOCKER_BIN} tag ${CONTAINER_REGISTRY_ID}/"${APP_NAME}" ${CONTAINER_REGISTRY_ID}/"${APP_NAME}":"${APP_VERSION}"
        JSON_APP=$(${DOCKER_BIN} images --format '{{json .}}' | jq ".| select(.Repository | contains(\"${IMAGE_FILTER}\")) |select(.Tag | contains(\"${APP_VERSION}\"))")
        APP_ID=$(echo "${JSON_APP}" | jq '.|.ID')
        echo "Info about your image :"
        echo "${JSON_APP}" | jq '.'
        echo "to try your container image locally :  ${DOCKER_BIN} run -p 8888:8080 --name ${APP_NAME} ${CONTAINER_REGISTRY_ID}/${APP_NAME}"
        echo "to try to open a shell inside your container :  ${DOCKER_BIN} run -it ${CONTAINER_REGISTRY_ID}/${APP_NAME} /bin/sh"
        echo "to deploy your container image to docker hub :  ${DOCKER_BIN} push ${CONTAINER_REGISTRY_ID}/${APP_NAME}"
        echo "to latter remove the images :  ${DOCKER_BIN} rmi ${CONTAINER_REGISTRY_ID}/${APP_NAME}"
      else
        echo "## 💥💥 ERROR: \"${IMAGE_FILTER}:${APP_VERSION}\" encountered a problem wile doing build !"
      fi
    else
      echo "You must correct the MEDIUM,HIGH,CRITICAL vulnerabilities detected by Trivy, before building your DockerFile" >&2
    fi
  else
      echo "## 💥💥 ERROR: \"${IMAGE_FILTER}:${APP_VERSION}\" this image version is already build !"
      echo "## 💥💥 ERROR: please upgrade version number in server.go file if you really want to rebuild !"
      echo "## 💥💥 ERROR: or remove the image with : ${DOCKER_BIN} rmi ${CONTAINER_REGISTRY_ID}/${APP_NAME}"
      echo "${JSON_APP}" | jq '.'
  fi
else
  echo "## 💥💥 ERROR: \"${DOCKER_BIN} is not available you should run rancher desktop"
fi
