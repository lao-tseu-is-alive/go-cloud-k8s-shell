#!/bin/bash
echo "##  Start a single instance of my ubuntu shell image"
kubectl run cg-shell  --image ghcr.io/lao-tseu-is-alive/go-cloud-k8s-shell:v0.1.7
echo "## Start an interactive shell access to my running ubuntu shell pod "
kubectl exec --stdin --tty cg-shell -- /bin/bash
