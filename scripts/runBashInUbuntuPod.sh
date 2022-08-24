#!/bin/bash
echo "about to enter an interactive session that will end after you exit the bash terminal"

# to scan for all ip in subnet :  nmap -n -sn 10.42.8.0/24 -oG - | awk '/Up$/{print $2}'
# to get a list of pods via API
# curl -s -k -H "Authorization: Bearer $TOKEN" -H 'Accept: application/json' $APISERVER/api/v1/pods |jq '.items[] | .metadata.namespace +":" + .metadata.name + ", ["  + .status.podIP + "]" + ", image:"  + .spec.containers[0].image + ", startTime:"  + .status.startTime '
kubectl -n default run -i --tty your-bash-session --image=ubuntu:22.04 --restart=Never -- bash
#next command will output a yaml file to create an pod like this
kubectl -n default run your-bash-session --image=ubuntu:22.04 --restart=Never bash --dry-run=client -o yaml
