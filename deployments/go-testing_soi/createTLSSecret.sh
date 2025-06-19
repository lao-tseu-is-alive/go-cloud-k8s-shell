#!/bin/bash
echo "##  Store your own TLS certificates for star.lausanne.ch in Kubernetes Secrets"
kubectl create secret tls go-cloud-k8s-shell-tls --cert=secret/comodo_2024.bundle --key=secret/comodo_lausanne_ch_2024_nopassword.key -n go-testing
