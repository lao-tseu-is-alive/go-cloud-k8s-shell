#!/bin/bash
echo "using bitnami kubeseal : A Kubernetes controller and tool for one-way encrypted Secrets"
echo "you need to follow instructions here : https://github.com/bitnami-labs/sealed-secrets/releases/"
kubeseal --format=yaml < app-secrets-go-cloud-k8s-shell.yaml  > sealed-app-secrets-go-cloud-k8s-shell.yaml
echo " now you can : kubectl apply -f sealed-app-secrets-go-cloud-k8s-shell.yaml"