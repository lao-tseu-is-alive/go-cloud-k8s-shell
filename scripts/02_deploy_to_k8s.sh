#!/bin/bash
DOCKER_BIN=docker
## Using nerdctl instead of docker on Linux, check: https://docs.rancherdesktop.io/images it's cool & ready to be used
DOCKER_BIN="nerdctl"
# uncomment next line if you are testing with rancher-desktop
#DOCKER_BIN="nerdctl -n k8s.io"
# you obviously will need to adjust next lines to your own favorite value :-)
CONTAINER_REGISTRY="ghcr.io/"
CONTAINER_REGISTRY_USER="lao-tseu-is-alive"
CONTAINER_REGISTRY_ID="${CONTAINER_REGISTRY}${CONTAINER_REGISTRY_USER}"
DEPLOYMENT_TEMPLATE="scripts/k8s-deployment_template.yml"
K8S_DEPLOYMENT=deployment.yml
echo "## Checking if ENV variable APP_NAME is already defined..."
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
echo "## USING APP_NAME: \"${APP_NAME}\", APP_VERSION: \"${APP_VERSION}\""
K8s_NAMESPACE="test-$APP_NAME"
IMAGE_FILTER="${CONTAINER_REGISTRY_ID}/${APP_NAME}"
echo "## Checking if image exist  ${IMAGE_FILTER} tag:v${APP_VERSION}"
JSON_APP=$(${DOCKER_BIN} images --format '{{json .}}' | jq ".| select(.Repository | contains(\"${IMAGE_FILTER}\")) |select(.Tag | contains(\"v${APP_VERSION}\"))")
APP_ID=$(echo "${JSON_APP}" | jq '.|.ID')
# checks whether APP_ID has length equal to zero --> meaning this image:version is not present and was probably not already build
if [[ -z "${APP_ID}" ]]
then
	echo "## ⚠️ ⚠️ WARNING: ${IMAGE_FILTER}:v${APP_VERSION} image was not found locally! will try to pull image from registry"
	echo "## about to run : ${DOCKER_BIN} pull ${IMAGE_FILTER}:v${APP_VERSION}"
	if ${DOCKER_BIN} pull "${IMAGE_FILTER}:v${APP_VERSION}" ;
	then
	  JSON_APP=$(${DOCKER_BIN} images --format '{{json .}}' | jq ".| select(.Repository | contains(\"${IMAGE_FILTER}\")) |select(.Tag | contains(\"v${APP_VERSION}\"))")
	  APP_ID=$(echo "${JSON_APP}" | jq '.|.ID')
	  echo "## ✓ ☺ 🚀 OK: \"${IMAGE_FILTER}:v${APP_VERSION}\" image was pulled successfully will prepare the deployment..."
    echo "${JSON_APP}" | jq '.'
	else
	  echo "## 💥💥 ERROR: ${IMAGE_FILTER}:v${APP_VERSION} image was not found in remote registry ! "
	  echo "## 💥💥 check if your image exist in https://github.com/${CONTAINER_REGISTRY_USER}/${APP_NAME}/pkgs/container/${APP_NAME}"
	  echo "## 💥💥 may be you should tag a new release an wait for github actions to build it ?"
	  exit
	fi
else
  echo "## ✓🚀 OK: \"${IMAGE_FILTER}:${APP_VERSION}\" image was found locally will prepare the deployment..."
  echo "${JSON_APP}" | jq '.'
fi
echo "## Generating a deployment based on template : ${DEPLOYMENT_TEMPLATE}"
DEPLOYMENT_DIRECTORY="deployments/${K8s_NAMESPACE}"
echo "## will store the deployment in directory    : ${DEPLOYMENT_DIRECTORY}"
mkdir "${DEPLOYMENT_DIRECTORY}"
echo "## will substitute APP_NAME : ${APP_NAME}"
sed s/APP_NAME/"${APP_NAME}"/g  ${DEPLOYMENT_TEMPLATE} > "${DEPLOYMENT_DIRECTORY}"/${K8S_DEPLOYMENT}
echo "## will substitute APP_VERSION : v${APP_VERSION}"
sed -i s/APP_VERSION/"v${APP_VERSION}"/g  "${DEPLOYMENT_DIRECTORY}"/${K8S_DEPLOYMENT}
echo "## will substitute GO_CONTAINER_REGISTRY_PREFIX : ${CONTAINER_REGISTRY_ID}"
sed -i "s|GO_CONTAINER_REGISTRY_PREFIX|${CONTAINER_REGISTRY_ID}|g"  "${DEPLOYMENT_DIRECTORY}"/${K8S_DEPLOYMENT}
echo "## Checking result of substitution in image name :"
yq  ".spec.template.spec.containers[0].image" "${DEPLOYMENT_DIRECTORY}"/${K8S_DEPLOYMENT}
#yq -i ".spec.template.spec.containers[0].image=\"${IMAGE_FILTER}:${APP_VERSION}\"" deployments/dev/deployment.yml
echo "## Checking for vulnerabilities with trivy"
cd "${DEPLOYMENT_DIRECTORY}" || exit
echo "## Checking for vulnerabilities in ${K8S_DEPLOYMENT}"
if trivy config --exit-code 1 --severity MEDIUM,HIGH,CRITICAL . ;
then
  echo "## ✓ ☺ 🚀 OK: Cool no vulnerabilities was found in your ${DEPLOYMENT}"
  cd "$OLDPWD" || exit
  # https://kubernetes.io/docs/tasks/access-application-cluster/list-all-running-container-images/
  # shellcheck disable=SC2021
  K8S_PODS_IMAGES=$(kubectl get pods -n "${K8s_NAMESPACE}" -o jsonpath="{..image}" | tr -s '[[:space:]]' '\n'|sort | uniq)
  if [[ $K8S_PODS_IMAGES =~ ${IMAGE_FILTER}:v${APP_VERSION} ]];
  then
    echo "## 💥💥 ERROR: ${IMAGE_FILTER}:v${APP_VERSION} image was already deployed ! "
     kubectl get pods -n "${K8s_NAMESPACE}" -l app="${APP_NAME}"
  fi
  #K8S_APP=$(kubectl get deployments --namespace=${K8s_NAMESPACE}" -o json |jq '.items[]?.spec.template.spec.containers[]?.image')
  echo "## Deploying ${K8S_DEPLOYMENT} in the K8S cluster in namespace ${K8s_NAMESPACE}"
  kubectl apply -f "${DEPLOYMENT_DIRECTORY}"/${K8S_DEPLOYMENT} --namespace="${K8s_NAMESPACE}"
  # Check deployment rollout status every 5 seconds (max 1 minutes) until complete.
  ATTEMPTS=0
  ROLLOUT_STATUS_CMD="kubectl rollout status deployment ${APP_NAME} --namespace=${K8s_NAMESPACE}"
  until $ROLLOUT_STATUS_CMD || [ $ATTEMPTS -eq 12 ]; do
    echo "## doing rollout status attempt num: ${ATTEMPTS} ..."
    $ROLLOUT_STATUS_CMD
    ATTEMPTS=$((ATTEMPTS + 1))
    sleep 5
  done
  echo "## Listing  pods in the cluster in namespace ${K8s_NAMESPACE}"
  kubectl get pods -o wide --namespace="${K8s_NAMESPACE}"
  echo "## Listing  services in the cluster "
  kubectl get service -o wide --namespace="${K8s_NAMESPACE}"
  #echo "## Listing  ingress in the cluster "
  #kubectl get ingress -o wide
  sleep 2
  #echo "## Running a curl on new service at cluster http://localhost:8000"
  #curl http://localhost:8000 | jq
  echo "## Running a curl on your service at : http://go-info-server.rancher.localhost:8000"
  curl -s http://go-info-server.rancher.localhost:8000 | jq
  echo "## you can later remove this deployment by running :"
  echo "kubectl delete -f ${DEPLOYMENT_DIRECTORY}/${K8S_DEPLOYMENT} --namespace=${K8s_NAMESPACE}"
  echo "## in case you have a pending in external ip for the get service it may be because a old daemonset is stil "
  echo "## check if there isn't an old daemonsets  with : kubectl  -n kube-system get daemonsets.apps "
  echo "## https://rancher.com/docs/k3s/latest/en/networking/#how-the-service-lb-works"
  # echo "Pods are allocated a private IP address by default and cannot be reached outside of the cluster unless you have a corresponding service."
  # echo "You can also use the kubectl port-forward command to map a local port to a port inside the pod like this : (ctrl+c to terminate)"
  # kubectl port-forward go-info-server-766947b78b-64f7j 8080:8080
else
  echo "## You must correct the MEDIUM,HIGH,CRITICAL vulnerabilities detected by Trivy, before building your DockerFile" >&2
fi

