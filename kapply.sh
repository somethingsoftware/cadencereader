#!/bin/bash
set -euo pipefail

kubectl apply -f k8s/
kubectl rollout restart deployment cadencereader 
kubectl rollout restart deployment dripper
