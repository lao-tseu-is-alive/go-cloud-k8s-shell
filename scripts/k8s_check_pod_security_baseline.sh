#!/bin/bash
kubectl label --dry-run=server --overwrite ns --all pod-security.kubernetes.io/enforce=baseline
