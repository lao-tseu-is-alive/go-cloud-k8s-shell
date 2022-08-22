#!/bin/bash
# https://kubernetes.io/docs/admin/authorization/rbac/
echo "about to bind default serviceaccount to allow request to api"
kubectl create clusterrole pod-reader --verb=get,list,watch --resource=pods
kubectl create clusterrolebinding pod-reader --clusterrole=pod-reader --serviceaccount=default:default
kubectl create clusterrole service-reader --verb=get,list,watch --resource=services
kubectl create clusterrolebinding service-reader --clusterrole=service-reader --serviceaccount=default:default