#!/bin/bash
echo "##  Start a single instance of my ubuntu shell image"
kubectl run cg-shell  --image ghcr.io/lao-tseu-is-alive/go-cloud-k8s-shell:v0.1.20 -n default
echo "## Start an interactive shell access to my running ubuntu shell pod "
kubectl exec --stdin --tty -n default cg-shell -- /bin/bash
