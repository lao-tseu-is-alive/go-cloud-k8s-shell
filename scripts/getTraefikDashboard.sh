#!/bin/bash
echo "## Exposing Traefik dashboard with port-forward on :  http://127.0.0.1:9000/dashboard/ "
kubectl port-forward -n kube-system $(kubectl -n kube-system get pods --selector "app.kubernetes.io/name=traefik" --output=name) 9000:9000
