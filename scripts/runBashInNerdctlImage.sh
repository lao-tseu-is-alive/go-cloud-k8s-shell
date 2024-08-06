#!/bin/bash
echo "about to enter an interactive session that will end after you exit the bash terminal"
nerdctl run  --rm -it --env-file .env --name go-cloud-k8s-shell  ghcr.io/lao-tseu-is-alive/go-cloud-k8s-shell /bin/bash
