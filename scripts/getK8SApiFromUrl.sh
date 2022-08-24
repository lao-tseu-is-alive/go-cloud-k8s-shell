#!/bin/bash
if [ -z "$1" ]
then
  URL="/api/v1/namespaces"
else
    URL="$1"
fi
APISERVER=https://kubernetes.default.svc
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt
export CACERT TOKEN NAMESPACE SERVICEACCOUNT APISERVER
#curl -s -k -H "Authorization: Bearer ${TOKEN}" -H 'Accept: application/json' ${APISERVER}"${URL}" |jq '.items[] | .metadata.namespace +":" + .metadata.name + ", ["  + .status.podIP + "]" + ", image:"  + .spec.containers[0].image + ", startTime:"  + .status.startTime '
curl -s --cacert ${CACERT} -H "Authorization: Bearer ${TOKEN}" -H 'Accept: application/json' ${APISERVER}"${URL}" |jq
