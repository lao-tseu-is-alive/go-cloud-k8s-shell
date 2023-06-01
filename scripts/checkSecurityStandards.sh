#!/bin/bash
echo "## Checking Pod Security Standards at the Cluster Level for all namespaces"
kubectl get namespaces
echo "## Checking privileged level"
kubectl label --dry-run=server --overwrite ns --all pod-security.kubernetes.io/enforce=privileged
echo "## Checking baseline level"
kubectl label --dry-run=server --overwrite ns --all pod-security.kubernetes.io/enforce=baseline
echo "## Checking restricted level"
kubectl label --dry-run=server --overwrite ns --all pod-security.kubernetes.io/enforce=restricted
echo "if you want to enforce restricted security standard level for the default namespace :"
echo "kubectl label  --overwrite ns default pod-security.kubernetes.io/enforce=restricted"
echo "more info : https://kubernetes.io/docs/tutorials/security/cluster-level-pss/"