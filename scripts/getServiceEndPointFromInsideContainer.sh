#!/bin/bash
TARGET_NAMESPACE=go-testing
TARGET_SERVICE=go-cloud-k8s-info-service
APISERVER=https://kubernetes.default.svc
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt
export CACERT TOKEN NAMESPACE SERVICEACCOUNT APISERVER
#TARGET_ENDPOINT=$(curl -s -k -H "Authorization: Bearer $TOKEN" -H 'Accept: application/json' $APISERVER/api/v1/namespaces/test-go-cloud-k8s-info/services |jq '.items[] | [.spec.clusterIP,  .spec.ports[0].targetPort|tostring ] | join(":")')
TARGET_ENDPOINT=$(curl -s -k -H "Authorization: Bearer $TOKEN" -H 'Accept: application/json' $APISERVER/api/v1/namespaces/$TARGET_NAMESPACE/services |jq --arg TARGET "${TARGET_SERVICE}" '.items[] | select (.metadata.name==$TARGET) | [.spec.clusterIP,  .spec.ports[0].targetPort|tostring ] | join(":")')
echo "### for ${TARGET_SERVICE} endpoint you can just try :  curl http://$TARGET_ENDPOINT"
echo "### or just : curl http://${TARGET_SERVICE}.${TARGET_NAMESPACE}:8000"
echo "LIST OF SERVICES IN CURRENT NAMESPACE (${TARGET_NAMESPACE}: "
curl -s -k -H "Authorization: Bearer $TOKEN" -H 'Accept: application/json' $APISERVER/api/v1/namespaces/$TARGET_NAMESPACE/services |jq '.items[] | .metadata.name'


