#!/bin/bash
APISERVER=https://kubernetes.default.svc
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt
export CACERT TOKEN NAMESPACE SERVICEACCOUNT APISERVER
wget -O - --no-check-certificate --header "Authorization: Bearer ${TOKEN}"  ${APISERVER}/api/v1
echo "you can run : wget -O - --no-check-certificate --header "Authorization: Bearer ${TOKEN}"  ${APISERVER}/api"
