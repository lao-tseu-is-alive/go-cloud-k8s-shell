# go-cloud-k8s-shell

[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=lao-tseu-is-alive_go-cloud-k8s-shell&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=lao-tseu-is-alive_go-cloud-k8s-shell)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=lao-tseu-is-alive_go-cloud-k8s-shell&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=lao-tseu-is-alive_go-cloud-k8s-shell)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=lao-tseu-is-alive_go-cloud-k8s-shell&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=lao-tseu-is-alive_go-cloud-k8s-shell)
[![cve-trivy-scan](https://github.com/lao-tseu-is-alive/go-cloud-k8s-shell/actions/workflows/cve-trivy-scan.yml/badge.svg)](https://github.com/lao-tseu-is-alive/go-cloud-k8s-shell/actions/workflows/cve-trivy-scan.yml)
[![codecov](https://codecov.io/gh/lao-tseu-is-alive/go-cloud-k8s-shell/branch/main/graph/badge.svg)](https://codecov.io/gh/lao-tseu-is-alive/go-cloud-k8s-shell)

A simple Golang microservice with basic tools to make some tests inside a k8s cluster

### some things to try inside a shell for this container:
 
you can check with curl another service already deployed like this : 

    curl "http://${GO_CLOUD_K8S_INFO_SERVICE_SERVICE_HOST}:${GO_CLOUD_K8S_INFO_SERVICE_SERVICE_PORT_HTTP}"

or with this shorter version :

    curl "http://go-cloud-k8s-info-service:${GO_CLOUD_K8S_INFO_SERVICE_SERVICE_PORT_HTTP}"

in those two previous example, we use the service exposed by the kubernetes yaml deployment in [go-cloud-k8s-info](https://github.com/lao-tseu-is-alive/go-cloud-k8s-info) 

you can also run :

    ./getServiceEndPointFromInsideContainer.sh

this script gives an example of service discovery using the kubernetes api


### More info :

+ [Checkov is a static code analysis tool for scanning infrastructure as code (IaC)](https://www.checkov.io/1.Welcome/What%20is%20Checkov.html)
+ [Kube-bench](https://github.com/aquasecurity/kube-bench/blob/main/docs/installation.md)
+ [Falco runtime security for hosts, containers, Kubernetes and the cloud.](https://falco.org/)