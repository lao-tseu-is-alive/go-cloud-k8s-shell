#!/bin/bash
echo "about to enter an interactive psql session that will end after you exit the bash terminal"

helm repo add bitnami https://charts.bitnami.com/bitnami
helm search repo bitnami
helm repo update
helm install mypostgresdb bitnami/postgresql

export POSTGRES_PASSWORD=$(kubectl get secret --namespace default mypostgresdb-postgresql -o jsonpath="{.data.postgres-password}" | base64 -d)
kubectl run mypostgresdb-postgresql-client --rm --tty -i --restart='Never' --namespace default --image docker.io/bitnami/postgresql:14.5.0-debian-11-r3 --env="PGPASSWORD=$POSTGRES_PASSWORD" \
      --command -- psql --host mypostgresdb-postgresql -U postgres -d postgres -p 5432
# you can rerieve the notes from this hel chart by running:
helm get notes mypostgresdb
