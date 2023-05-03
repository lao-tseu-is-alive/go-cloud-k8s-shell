#!/bin/bash
curl "http://go-cloud-k8s-info-service.${MY_POD_NAMESPACE}:${GO_CLOUD_K8S_INFO_SERVICE_SERVICE_PORT}" | jq