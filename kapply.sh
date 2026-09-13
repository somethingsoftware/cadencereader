#!/bin/bash
set -euo pipefail

kubectl apply -f k8s/
kubectl rollout restart deployment cadencereader 
kubectl delete job immediate-rssimport # in case it already exists
kubectl create job immediate-rssimport --from=cronjob/rssimport
