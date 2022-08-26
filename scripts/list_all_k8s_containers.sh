#!/bin/bash
kubectl get pods --all-namespaces -o jsonpath="{.items[*].spec.containers[*].image}" |tr -s '[[:space:]]' '\n' |sort |uniq -c
trivy --severity MEDIUM,HIGH,CRITICAL image ghcr.io/lao-tseu-is-alive/go-cloud-k8s-shell:v0.1.5